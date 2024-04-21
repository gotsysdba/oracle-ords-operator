/*
** Copyright (c) 2024 Oracle and/or its affiliates.
**
** The Universal Permissive License (UPL), Version 1.0
**
** Subject to the condition set forth below, permission is hereby granted to any
** person obtaining a copy of this software, associated documentation and/or data
** (collectively the "Software"), free of charge and under any and all copyright
** rights in the Software, and any and all patent rights owned or freely
** licensable by each licensor hereunder covering either (i) the unmodified
** Software as contributed to or provided by such licensor, or (ii) the Larger
** Works (as defined below), to deal in both
**
** (a) the Software, and
** (b) any piece of software and/or hardware listed in the lrgrwrks.txt file if
** one is included with the Software (each a "Larger Work" to which the Software
** is contributed by such licensors),
**
** without restriction, including without limitation the rights to copy, create
** derivative works of, display, perform, and distribute the Software and make,
** use, sell, offer for sale, import, export, have made, and have sold the
** Software and the Larger Work(s), and to sublicense the foregoing rights on
** either these or other terms.
**
** This license is subject to the following condition:
** The above copyright notice and either this complete permission notice or at
** a minimum a reference to the UPL must be included in all copies or
** substantial portions of the Software.
**
** THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
** IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
** FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
** AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
** LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
** OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
** SOFTWARE.
 */

package controller

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	databasev1 "example.com/oracle-ords-operator/api/v1"
)

const RestDataServicesFinalizer = "database.oracle.com/restdataservicesfinalizer"

// Definitions to manage status conditions
const (
	typeAvailable = "Available"
	typeModified  = "Modified" //requires pod restart
	typeDegraded  = "Degraded"
)

// Definitions of Standards
const (
	ordsConfigBase       = "/opt/oracle/sa/config"
	servicePortName      = "sa-svc-port"
	targetPortName       = "sa-pod-port"
	globalConfigName     = "sa-settings-global"
	poolConfigPreName    = "sa-settings-" // Append PoolName
	poolComponentLabel   = "sa-pool-setting"
	globalComponentLabel = "sa-global-setting"
)

// RestDataServicesReconciler reconciles a RestDataServices object
type RestDataServicesReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=database.oracle.com,resources=restdataservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=database.oracle.com,resources=restdataservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=database.oracle.com,resources=restdataservices/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=secrets/status,verbs=get
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=deployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=daemonsets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=statefulsets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

func (r *RestDataServicesReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logr := log.FromContext(ctx)
	ords := &databasev1.RestDataServices{}

	// Check if there is an ORDS resource; if not nothing to reconcile
	if err := r.Get(ctx, req.NamespacedName, ords); err != nil {
		if apierrors.IsNotFound(err) {
			logr.Info("Resource Deleted")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{Requeue: true, RequeueAfter: time.Minute}, err
	}

	// ConfigMaps
	result, err := r.ConfigMapReconcile(ctx, req, ords)
	if result.Requeue || err != nil {
		logr.Info("Returning from ConfigMapReconcile")
		return result, err
	}

	// Workloads
	switch ords.Spec.WorkloadType {
	case "StatefulSet":
		result, err = r.StatefulSetReconcile(ctx, req, ords)
		if result.Requeue || err != nil {
			return result, err
		}
	case "DaemonSet":
		result, err = r.DaemonSetReconcile(ctx, req, ords)
		if result.Requeue || err != nil {
			return result, err
		}
	default:
		result, err = r.DeploymentReconcile(ctx, req, ords)
		if result.Requeue || err != nil {
			return result, err
		}
	}
	result, err = r.WorkloadReconcile(ctx, req, ords)

	// Service
	result, err = r.ServiceReconcile(ctx, req, ords)
	if result.Requeue || err != nil {
		return result, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RestDataServicesReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&databasev1.RestDataServices{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&appsv1.DaemonSet{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

/************************************************
* ConfigMaps
*************************************************/
func (r *RestDataServicesReconciler) ConfigMapReconcile(ctx context.Context, req ctrl.Request, ords *databasev1.RestDataServices) (ctrl.Result, error) {
	logr := log.FromContext(ctx).WithName("ConfigMapReconcile")

	configMapType := &corev1.ConfigMap{}
	// Global
	err := r.Get(ctx, types.NamespacedName{Name: globalConfigName, Namespace: ords.Namespace}, configMapType)
	if err != nil && apierrors.IsNotFound(err) {
		def, err := r.defGlobalConfigMap(ctx, ords)
		if err = r.Create(ctx, def); err != nil {
			return ctrl.Result{}, err
		}
		logr.Info("Created: " + globalConfigName)
	} else {
		newGlobalConfigMap, err := r.defGlobalConfigMap(ctx, ords)
		if err == nil && !equality.Semantic.DeepEqual(configMapType.Data, newGlobalConfigMap.Data) {
			if err := r.Update(ctx, newGlobalConfigMap); err != nil {
				return ctrl.Result{}, err
			}
			logr.Info("Reconciled: " + globalConfigName)
		}
	}

	// Pools
	definedPools := make(map[string]bool)
	for i := 0; i < len(ords.Spec.PoolSettings); i++ {
		poolName := ords.Spec.PoolSettings[i].PoolName
		poolConfigMapName := poolConfigPreName + strings.ToLower(poolName)
		definedPools[poolConfigMapName] = true
		err = r.Get(ctx, types.NamespacedName{Name: poolConfigMapName, Namespace: ords.Namespace}, configMapType)
		if err != nil && apierrors.IsNotFound(err) {
			def, err := r.defPoolConfigMap(ctx, ords, poolConfigMapName, i)
			if err != nil {
				return ctrl.Result{}, err
			}
			if err = r.Create(ctx, def); err != nil {
				return ctrl.Result{}, err
			}
			logr.Info("Created: " + poolConfigMapName)
		} else {
			newPoolConfigMap, err := r.defPoolConfigMap(ctx, ords, poolConfigMapName, i)
			if err == nil && !equality.Semantic.DeepEqual(configMapType.Data, newPoolConfigMap.Data) {
				if err := r.Update(ctx, newPoolConfigMap); err != nil {
					return ctrl.Result{}, err
				}
				logr.Info("Reconciled: " + poolConfigMapName)
			}
		}
	}

	// Delete undefined pools
	configMapList := &corev1.ConfigMapList{}
	if err := r.List(ctx, configMapList, client.InNamespace(req.Namespace),
		client.MatchingLabels(map[string]string{"app.kubernetes.io/component": poolComponentLabel}),
	); err != nil {
		return ctrl.Result{}, err
	}
	for _, configMapType := range configMapList.Items {
		if _, exists := definedPools[configMapType.Name]; !exists {
			if err := r.Delete(ctx, &configMapType); err != nil {
				return ctrl.Result{}, err
			}
			logr.Info("Deleted: " + configMapType.Name)
		}
	}
	return ctrl.Result{}, nil
}

/************************************************
* Workloads
*************************************************/
func (r *RestDataServicesReconciler) DeploymentReconcile(ctx context.Context, req ctrl.Request, ords *databasev1.RestDataServices) (ctrl.Result, error) {
	logr := log.FromContext(ctx).WithName("DeploymentReconcile")
	deploymentType := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: ords.Name, Namespace: ords.Namespace}, deploymentType)
	if err != nil && apierrors.IsNotFound(err) {
		def, err := r.defDeployment(ctx, ords)
		if err = r.Create(ctx, def); err != nil {
			return ctrl.Result{}, err
		}
		logr.Info("Created: " + ords.Name)
		return ctrl.Result{}, nil
	}

	definedReplicas := ords.Spec.Replicas
	if *deploymentType.Spec.Replicas != definedReplicas {
		deploymentType.Spec.Replicas = &definedReplicas
		if err := r.Update(ctx, deploymentType); err != nil {
			return ctrl.Result{}, err
		}
		logr.Info("Scaled")
	}
	return ctrl.Result{}, nil
}

func (r *RestDataServicesReconciler) StatefulSetReconcile(ctx context.Context, req ctrl.Request, ords *databasev1.RestDataServices) (ctrl.Result, error) {
	logr := log.FromContext(ctx).WithName("StatefulSetReconcile")
	statefulSetType := &appsv1.StatefulSet{}
	err := r.Get(ctx, types.NamespacedName{Name: ords.Name, Namespace: ords.Namespace}, statefulSetType)
	if err != nil && apierrors.IsNotFound(err) {
		def, err := r.defStatefulSet(ctx, ords)
		if err = r.Create(ctx, def); err != nil {
			return ctrl.Result{}, err
		}
		logr.Info("Created: " + ords.Name)
		return ctrl.Result{}, nil
	}

	definedReplicas := ords.Spec.Replicas
	if *statefulSetType.Spec.Replicas != definedReplicas {
		statefulSetType.Spec.Replicas = &definedReplicas
		if err := r.Update(ctx, statefulSetType); err != nil {
			return ctrl.Result{}, err
		}
		logr.Info("Scaled")
	}
	return ctrl.Result{}, nil
}

func (r *RestDataServicesReconciler) DaemonSetReconcile(ctx context.Context, req ctrl.Request, ords *databasev1.RestDataServices) (ctrl.Result, error) {
	logr := log.FromContext(ctx).WithName("DaemonSetReconcile")
	daemonSetType := &appsv1.DaemonSet{}
	err := r.Get(ctx, types.NamespacedName{Name: ords.Name, Namespace: ords.Namespace}, daemonSetType)
	if err != nil && apierrors.IsNotFound(err) {
		def, err := r.defDaemonSet(ctx, ords)
		if err = r.Create(ctx, def); err != nil {
			return ctrl.Result{}, err
		}
		logr.Info("Created: " + ords.Name)
	}
	return ctrl.Result{}, nil
}

func (r *RestDataServicesReconciler) WorkloadReconcile(ctx context.Context, req ctrl.Request, ords *databasev1.RestDataServices) (ctrl.Result, error) {
	deployErr := error(nil)
	dsErr := error(nil)
	stsErr := error(nil)
	switch ords.Spec.WorkloadType {
	case "StatefulSet":
		deployErr = r.DeleteDeployment(ctx, req, ords)
		dsErr = r.DeleteDaemonSet(ctx, req, ords)
	case "DaemonSet":
		deployErr = r.DeleteDeployment(ctx, req, ords)
		stsErr = r.DeleteStatefulSet(ctx, req, ords)
	default:
		dsErr = r.DeleteDaemonSet(ctx, req, ords)
		stsErr = r.DeleteStatefulSet(ctx, req, ords)
	}
	if deployErr != nil || dsErr != nil || stsErr != nil {
		return ctrl.Result{}, errors.New("unable to cleanup workloads")
	}
	return ctrl.Result{}, nil
}

func (r *RestDataServicesReconciler) DeleteDeployment(ctx context.Context, req ctrl.Request, ords *databasev1.RestDataServices) error {
	logr := log.FromContext(ctx).WithName("DeleteDeployment")
	deploymentList := &appsv1.DeploymentList{}
	if err := r.List(ctx, deploymentList, client.InNamespace(req.Namespace),
		client.MatchingLabels(map[string]string{"app.kubernetes.io/component": "workload"}),
	); err != nil {
		return err
	}
	for _, deploymentType := range deploymentList.Items {
		if err := r.Delete(ctx, &deploymentType); err != nil {
			return err
		}
		logr.Info("Deleted: " + deploymentType.Name)
	}
	return nil
}

func (r *RestDataServicesReconciler) DeleteStatefulSet(ctx context.Context, req ctrl.Request, ords *databasev1.RestDataServices) error {
	logr := log.FromContext(ctx).WithName("StatefulSet")
	statefulSetList := &appsv1.StatefulSetList{}
	if err := r.List(ctx, statefulSetList, client.InNamespace(req.Namespace),
		client.MatchingLabels(map[string]string{"app.kubernetes.io/component": "workload"}),
	); err != nil {
		return err
	}
	for _, statefulSetType := range statefulSetList.Items {
		if err := r.Delete(ctx, &statefulSetType); err != nil {
			return err
		}
		logr.Info("Deleted: " + statefulSetType.Name)
	}
	return nil
}

func (r *RestDataServicesReconciler) DeleteDaemonSet(ctx context.Context, req ctrl.Request, ords *databasev1.RestDataServices) error {
	logr := log.FromContext(ctx).WithName("DeleteDaemonSet")
	daemonSetList := &appsv1.DaemonSetList{}
	if err := r.List(ctx, daemonSetList, client.InNamespace(req.Namespace),
		client.MatchingLabels(map[string]string{"app.kubernetes.io/component": "workload"}),
	); err != nil {
		return err
	}
	for _, daemonSetType := range daemonSetList.Items {
		if err := r.Delete(ctx, &daemonSetType); err != nil {
			return err
		}
		logr.Info("Deleted: " + daemonSetType.Name)
	}
	return nil
}

// Service
func (r *RestDataServicesReconciler) ServiceReconcile(ctx context.Context, req ctrl.Request, ords *databasev1.RestDataServices) (ctrl.Result, error) {
	logr := log.FromContext(ctx).WithName("ServiceReconcile")

	serviceType := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: ords.Name, Namespace: ords.Namespace}, serviceType)
	if err != nil && apierrors.IsNotFound(err) {
		def, err := r.defService(ctx, ords)
		if err = r.Create(ctx, def); err != nil {
			return ctrl.Result{}, err
		}
		logr.Info("Created: " + ords.Name)
		return ctrl.Result{}, nil
	}

	definedServicePort := int32(80)
	if ords.Spec.GlobalSettings.StandaloneHttpPort != nil {
		definedServicePort = *ords.Spec.GlobalSettings.StandaloneHttpPort
	}
	for _, existingPort := range serviceType.Spec.Ports {
		if existingPort.Name == servicePortName {
			if existingPort.Port != definedServicePort {
				if err := r.Update(ctx, serviceType); err != nil {
					return ctrl.Result{}, err
				}
				logr.Info("Reconciled: " + existingPort.Name)
			}
			return ctrl.Result{}, nil
		}
	}
	return ctrl.Result{}, nil
}

/*************************************************
* Definers
/************************************************/
// Global ConfigMap
func (r *RestDataServicesReconciler) defGlobalConfigMap(ctx context.Context, ords *databasev1.RestDataServices) (*corev1.ConfigMap, error) {
	labels := getLabels(ords.Name, globalComponentLabel)
	def := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      globalConfigName,
			Namespace: ords.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"settings.xml": fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
				`<!DOCTYPE properties SYSTEM "http://java.sun.com/dtd/properties.dtd">` + "\n" +
				`<properties>` + "\n" +
				// `  <entry key="standalone.https.cert">/opt/oracle/sa/config/certficate</entry>` + "\n" +
				// `  <entry key="standalone.https.cert.key">/opt/oracle/sa/config/certficate</entry>` + "\n" +
				conditionalEntry("cache.metadata.graphql.expireAfterAccess", ords.Spec.GlobalSettings.CacheMetadataGraphqlExpireAfterAccess) +
				conditionalEntry("cache.metadata.jwks.enabled", ords.Spec.GlobalSettings.CacheMetadataJwksEnabled) +
				conditionalEntry("cache.metadata.jwks.initialCapacity", ords.Spec.GlobalSettings.CacheMetadataJwksInitialCapacity) +
				conditionalEntry("cache.metadata.jwks.maximumSize", ords.Spec.GlobalSettings.CacheMetadataJwksMaximumSize) +
				conditionalEntry("cache.metadata.jwks.expireAfterAccess", ords.Spec.GlobalSettings.CacheMetadataJwksExpireAfterAccess) +
				conditionalEntry("cache.metadata.jwks.expireAfterWrite", ords.Spec.GlobalSettings.CacheMetadataJwksExpireAfterWrite) +
				conditionalEntry("database.api.management.services.disabled", ords.Spec.GlobalSettings.DatabaseApiManagementServicesDisabled) +
				conditionalEntry("db.invalidPoolTimeout", ords.Spec.GlobalSettings.DbInvalidPoolTimeout) +
				conditionalEntry("feature.grahpql.max.nesting.depth", ords.Spec.GlobalSettings.FeatureGrahpqlMaxNestingDepth) +
				conditionalEntry("request.traceHeaderName", ords.Spec.GlobalSettings.RequestTraceHeaderName) +
				conditionalEntry("security.credentials.attempts ", ords.Spec.GlobalSettings.SecurityCredentialsAttempts) +
				conditionalEntry("security.credentials.file ", ords.Spec.GlobalSettings.SecurityCredentialsFile) +
				conditionalEntry("security.credentials.lock.time ", ords.Spec.GlobalSettings.SecurityCredentialsLockTime) +
				conditionalEntry("standalone.access.log", ords.Spec.GlobalSettings.StandaloneAccessLog) +
				conditionalEntry("standalone.binds", ords.Spec.GlobalSettings.StandaloneBinds) +
				conditionalEntry("standalone.context.path ", ords.Spec.GlobalSettings.StandaloneContextPath) +
				conditionalEntry("standalone.doc.root", ords.Spec.GlobalSettings.StandaloneDocRoot) +
				conditionalEntry("standalone.http.port", ords.Spec.GlobalSettings.StandaloneHttpPort) +
				conditionalEntry("standalone.https.host", ords.Spec.GlobalSettings.StandaloneHttpsHost) +
				conditionalEntry("standalone.https.port", ords.Spec.GlobalSettings.StandaloneHttpsPort) +
				conditionalEntry("standalone.static.context.path ", ords.Spec.GlobalSettings.StandaloneStaticContextPath) +
				conditionalEntry("standalone.static.path", ords.Spec.GlobalSettings.StandaloneStaticPath) +
				conditionalEntry("standalone.stop.timeout ", ords.Spec.GlobalSettings.StandaloneStopTimeout) +
				conditionalEntry("cache.metadata.timeout", ords.Spec.GlobalSettings.CacheMetadataTimeout) +
				conditionalEntry("cache.metadata.enabled", ords.Spec.GlobalSettings.CacheMetadataEnabled) +
				conditionalEntry("database.api.enabled", ords.Spec.GlobalSettings.DatabaseApiEnabled) +
				conditionalEntry("debug.printDebugToScreen", ords.Spec.GlobalSettings.DebugPrintDebugToScreen) +
				conditionalEntry("error.responseFormat", ords.Spec.GlobalSettings.ErrorResponseFormat) +
				conditionalEntry("error.externalPath", ords.Spec.GlobalSettings.ErrorExternalPath) +
				conditionalEntry("icap.port", ords.Spec.GlobalSettings.IcapPort) +
				conditionalEntry("icap.secure.port", ords.Spec.GlobalSettings.IcapSecurePort) +
				conditionalEntry("icap.server", ords.Spec.GlobalSettings.IcapServer) +
				conditionalEntry("log.procedure", ords.Spec.GlobalSettings.LogProcedure) +
				conditionalEntry("security.disableDefaultExclusionList", ords.Spec.GlobalSettings.SecurityDisableDefaultExclusionList) +
				conditionalEntry("security.exclusionList", ords.Spec.GlobalSettings.SecurityExclusionList) +
				conditionalEntry("security.inclusionList", ords.Spec.GlobalSettings.SecurityInclusionList) +
				conditionalEntry("security.maxEntries", ords.Spec.GlobalSettings.SecurityMaxEntries) +
				conditionalEntry("security.verifySSL", ords.Spec.GlobalSettings.SecurityVerifySSL) +
				`</properties>`),
		},
	}

	// Set the ownerRef
	if err := ctrl.SetControllerReference(ords, def, r.Scheme); err != nil {
		return nil, err
	}
	return def, nil
}

// Pool ConfigMaps
func (r *RestDataServicesReconciler) defPoolConfigMap(ctx context.Context, ords *databasev1.RestDataServices, poolConfigName string, i int) (*corev1.ConfigMap, error) {
	labels := getLabels(ords.Name, poolComponentLabel)

	def := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      poolConfigName,
			Namespace: ords.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"pool.xml": fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
				`<!DOCTYPE properties SYSTEM "http://java.sun.com/dtd/properties.dtd">` + "\n" +
				`<properties>` + "\n" +
				`  <entry key="db.username">` + ords.Spec.PoolSettings[i].DbUsername + `</entry>` + "\n" +
				conditionalEntry("db.adminUser", ords.Spec.PoolSettings[i].DbAdminUser) +
				conditionalEntry("db.cdb.adminUser", ords.Spec.PoolSettings[i].DbCdbAdminUser) +
				conditionalEntry("apex.security.administrator.roles", ords.Spec.PoolSettings[i].ApexSecurityAdministratorRoles) +
				conditionalEntry("apex.security.user.roles", ords.Spec.PoolSettings[i].ApexSecurityUserRoles) +
				conditionalEntry("autoupgrade.api.aulocation", ords.Spec.PoolSettings[i].AutoupgradeApiAulocation) +
				conditionalEntry("autoupgrade.api.enabled", ords.Spec.PoolSettings[i].AutoupgradeApiEnabled) +
				conditionalEntry("autoupgrade.api.jvmlocation", ords.Spec.PoolSettings[i].AutoupgradeApiJvmlocation) +
				conditionalEntry("autoupgrade.api.loglocation", ords.Spec.PoolSettings[i].AutoupgradeApiLoglocation) +
				conditionalEntry("db.credentialsSource", ords.Spec.PoolSettings[i].DbCredentialsSource) +
				conditionalEntry("db.poolDestroyTimeout", ords.Spec.PoolSettings[i].DbPoolDestroyTimeout) +
				conditionalEntry("db.wallet.zip", ords.Spec.PoolSettings[i].DbWalletZip) +
				conditionalEntry("db.wallet.zip.path", ords.Spec.PoolSettings[i].DbWalletZipPath) +
				conditionalEntry("db.wallet.zip.service", ords.Spec.PoolSettings[i].DbWalletZipService) +
				conditionalEntry("debug.trackResources", ords.Spec.PoolSettings[i].DebugTrackResources) +
				conditionalEntry("feature.openservicebroker.exclude", ords.Spec.PoolSettings[i].FeatureOpenservicebrokerExclude) +
				conditionalEntry("feature.sdw", ords.Spec.PoolSettings[i].FeatureSdw) +
				conditionalEntry("http.cookie.filter", ords.Spec.PoolSettings[i].HttpCookieFilter) +
				conditionalEntry("jdbc.auth.admin.role", ords.Spec.PoolSettings[i].JdbcAuthAdminRole) +
				conditionalEntry("jdbc.cleanup.mode", ords.Spec.PoolSettings[i].JdbCleanupMode) +
				conditionalEntry("owa.trace.sql", ords.Spec.PoolSettings[i].OwaTraceSql) +
				conditionalEntry("plsql.gateway.mode", ords.Spec.PoolSettings[i].PlsqlGatewayMode) +
				conditionalEntry("security.jwt.profile.enabled", ords.Spec.PoolSettings[i].SecurityJwtProfileEnabled) +
				conditionalEntry("security.jwks.size", ords.Spec.PoolSettings[i].SecurityJwksSize) +
				conditionalEntry("security.jwks.connection.timeout", ords.Spec.PoolSettings[i].SecurityJwksConnectionTimeout) +
				conditionalEntry("security.jwks.read.timeout", ords.Spec.PoolSettings[i].SecurityJwksReadTimeout) +
				conditionalEntry("security.jwks.refresh.interval", ords.Spec.PoolSettings[i].SecurityJwksRefreshInterval) +
				conditionalEntry("security.jwt.allowed.skew", ords.Spec.PoolSettings[i].SecurityJwtAllowedSkew) +
				conditionalEntry("security.jwt.allowed.age", ords.Spec.PoolSettings[i].SecurityJwtAllowedAge) +
				conditionalEntry("security.jwt.allowed.age", ords.Spec.PoolSettings[i].SecurityValidationFunctionType) +
				conditionalEntry("db.connectionType", ords.Spec.PoolSettings[i].DbConnectionType) +
				conditionalEntry("db.customURL", ords.Spec.PoolSettings[i].DbCustomURL) +
				conditionalEntry("db.hostname", ords.Spec.PoolSettings[i].DbHostname) +
				conditionalEntry("db.port", ords.Spec.PoolSettings[i].DbPort) +
				conditionalEntry("db.servicename", ords.Spec.PoolSettings[i].DbServicename) +
				conditionalEntry("db.serviceNameSuffix", ords.Spec.PoolSettings[i].DbServiceNameSuffix) +
				conditionalEntry("db.sid", ords.Spec.PoolSettings[i].DbSid) +
				conditionalEntry("db.tnsAliasName", ords.Spec.PoolSettings[i].DbTnsAliasName) +
				conditionalEntry("db.tnsDirectory", ords.Spec.PoolSettings[i].DbTnsDirectory) +
				conditionalEntry("jdbc.DriverType", ords.Spec.PoolSettings[i].JdbcDriverType) +
				conditionalEntry("jdbc.InactivityTimeout", ords.Spec.PoolSettings[i].JdbcInactivityTimeout) +
				conditionalEntry("jdbc.InitialLimit", ords.Spec.PoolSettings[i].JdbcInitialLimit) +
				conditionalEntry("jdbc.MaxConnectionReuseCount", ords.Spec.PoolSettings[i].JdbcMaxConnectionReuseCount) +
				conditionalEntry("jdbc.MaxLimit", ords.Spec.PoolSettings[i].JdbcMaxLimit) +
				conditionalEntry("jdbc.auth.enabled", ords.Spec.PoolSettings[i].JdbcAuthEnabled) +
				conditionalEntry("jdbc.MaxStatementsLimit", ords.Spec.PoolSettings[i].JdbcMaxStatementsLimit) +
				conditionalEntry("jdbc.MinLimit", ords.Spec.PoolSettings[i].JdbcMinLimit) +
				conditionalEntry("jdbc.statementTimeout", ords.Spec.PoolSettings[i].JdbcStatementTimeout) +
				conditionalEntry("misc.defaultPage", ords.Spec.PoolSettings[i].MiscDefaultPage) +
				conditionalEntry("misc.pagination.maxRows", ords.Spec.PoolSettings[i].MiscPaginationMaxRows) +
				conditionalEntry("procedure.postProcess", ords.Spec.PoolSettings[i].ProcedurePostProcess) +
				conditionalEntry("procedure.preProcess", ords.Spec.PoolSettings[i].ProcedurePreProcess) +
				conditionalEntry("procedure.rest.preHook", ords.Spec.PoolSettings[i].ProcedureRestPreHook) +
				conditionalEntry("security.requestAuthenticationFunction", ords.Spec.PoolSettings[i].SecurityRequestAuthenticationFunction) +
				conditionalEntry("security.requestValidationFunction", ords.Spec.PoolSettings[i].SecurityRequestValidationFunction) +
				conditionalEntry("soda.defaultLimit", ords.Spec.PoolSettings[i].SodaDefaultLimit) +
				conditionalEntry("soda.maxLimit", ords.Spec.PoolSettings[i].SodaMaxLimit) +
				conditionalEntry("restEnabledSql.active", ords.Spec.PoolSettings[i].RestEnabledSqlActive) +
				`</properties>`),
		},
	}

	// Set the ownerRef
	if err := ctrl.SetControllerReference(ords, def, r.Scheme); err != nil {
		return nil, err
	}
	return def, nil
}

// Workloads
func (r *RestDataServicesReconciler) defDeployment(ctx context.Context, ords *databasev1.RestDataServices) (*appsv1.Deployment, error) {
	labels := getLabels(ords.Name, "workload")
	replicas := ords.Spec.Replicas
	podTemplate := defPods(ords)

	def := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ords.Name,
			Namespace: ords.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: podTemplate,
			},
		},
	}

	// Set the ownerRef
	if err := ctrl.SetControllerReference(ords, def, r.Scheme); err != nil {
		return nil, err
	}
	return def, nil
}

func (r *RestDataServicesReconciler) defStatefulSet(ctx context.Context, ords *databasev1.RestDataServices) (*appsv1.StatefulSet, error) {
	labels := getLabels(ords.Name, "workload")
	replicas := ords.Spec.Replicas
	podTemplate := defPods(ords)

	def := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ords.Name,
			Namespace: ords.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: podTemplate,
			},
		},
	}

	// Set the ownerRef
	if err := ctrl.SetControllerReference(ords, def, r.Scheme); err != nil {
		return nil, err
	}
	return def, nil
}

func (r *RestDataServicesReconciler) defDaemonSet(ctx context.Context, ords *databasev1.RestDataServices) (*appsv1.DaemonSet, error) {
	labels := getLabels(ords.Name, "workload")
	podTemplate := defPods(ords)
	def := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ords.Name,
			Namespace: ords.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: podTemplate,
			},
		},
	}

	// Set the ownerRef
	if err := ctrl.SetControllerReference(ords, def, r.Scheme); err != nil {
		return nil, err
	}
	return def, nil
}

// Pods
func defPods(ords *databasev1.RestDataServices) corev1.PodSpec {
	specVolumes, specVolumeMounts := defVolumes(ords)
	port := int32(8080)
	if ords.Spec.GlobalSettings.StandaloneHttpPort != nil {
		port = *ords.Spec.GlobalSettings.StandaloneHttpPort
	}

	podTemplate := corev1.PodSpec{
		Volumes: specVolumes,
		SecurityContext: &corev1.PodSecurityContext{
			RunAsNonRoot: &[]bool{true}[0],
			FSGroup:      &[]int64{54321}[0],
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeRuntimeDefault,
			},
		},
		Containers: []corev1.Container{{
			Image:           ords.Spec.Image,
			Name:            ords.Name,
			ImagePullPolicy: corev1.PullIfNotPresent,
			SecurityContext: &corev1.SecurityContext{
				RunAsNonRoot:             &[]bool{true}[0],
				RunAsUser:                &[]int64{54321}[0],
				AllowPrivilegeEscalation: &[]bool{false}[0],
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{
						"ALL",
					},
				},
			},
			Ports: []corev1.ContainerPort{{
				ContainerPort: port,
				Name:          targetPortName,
			}},
			Command: []string{"sh", "-c", "tail -f /dev/null"},
			//Command: []string{"/bin/bash", "-c", "ords --config $ORDS_CONFIG serve"},
			Env: []corev1.EnvVar{
				{
					Name:  "ORDS_CONFIG",
					Value: ordsConfigBase,
				},
			},
			VolumeMounts: specVolumeMounts,
		}},
	}
	return podTemplate
}

// Volumes
func defVolumes(ords *databasev1.RestDataServices) ([]corev1.Volume, []corev1.VolumeMount) {
	// Initialize the slice to hold specifications
	var volumes []corev1.Volume
	var volumeMounts []corev1.VolumeMount

	// Build volume specifications for globalSettings
	standaloneVolume := buildVolume("standalone", "EmptyDir")
	globalWalletVolume := buildVolume("sa-wallet-global", "EmptyDir")
	globalConfigVolume := buildVolume(globalConfigName, "ConfigMap")
	volumes = append(volumes, standaloneVolume, globalWalletVolume, globalConfigVolume)

	standaloneVolumeMount := buildVolumeMount("standalone", ordsConfigBase+"/global/standalone/", false)
	globalWalletVolumeMount := buildVolumeMount("sa-wallet-global", ordsConfigBase+"/global/wallet/", false)
	globalConfigVolumeMount := buildVolumeMount(globalConfigName, ordsConfigBase+"/global/", false)
	volumeMounts = append(volumeMounts, standaloneVolumeMount, globalWalletVolumeMount, globalConfigVolumeMount)

	// Build volume specifications for each pool in poolSettings
	for _, pool := range ords.Spec.PoolSettings {
		poolName := strings.ToLower(pool.PoolName)
		poolConfigName := poolConfigPreName + poolName
		poolWalletName := "sa-wallet-" + poolName
		// Volumes
		poolWalletVolume := buildVolume(poolWalletName, "EmptyDir")
		poolConfigVolume := buildVolume(poolConfigName, "ConfigMap")

		volumes = append(volumes, poolWalletVolume, poolConfigVolume)
		// VolumeMounts
		poolWalletVolumeMount := buildVolumeMount(poolWalletName, ordsConfigBase+"/databases/"+poolName+"/wallet", false)
		poolConfigVolumeMount := buildVolumeMount(poolConfigName, ordsConfigBase+"/databases/"+poolName+"/", false)
		volumeMounts = append(volumeMounts, poolWalletVolumeMount, poolConfigVolumeMount)
	}
	return volumes, volumeMounts
}

func buildVolumeMount(name string, path string, readOnly bool) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      name,
		MountPath: path,
		ReadOnly:  readOnly,
	}
}

func buildVolume(name string, source string) corev1.Volume {
	switch source {
	case "ConfigMap":
		return corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: name,
					},
				},
			},
		}
	case "EmptyDir":
		return corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}
	default:
		return corev1.Volume{}
	}
}

// Service
func (r *RestDataServicesReconciler) defService(ctx context.Context, ords *databasev1.RestDataServices) (*corev1.Service, error) {
	port := int32(80)
	if ords.Spec.GlobalSettings.StandaloneHttpPort != nil {
		port = *ords.Spec.GlobalSettings.StandaloneHttpPort
	}
	labels := getLabels(ords.Name, "service")
	def := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ords.Name,
			Namespace: ords.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       servicePortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       port,
					TargetPort: intstr.FromString(targetPortName),
				},
			},
		},
	}
	if err := ctrl.SetControllerReference(ords, def, r.Scheme); err != nil {
		return nil, err
	}
	return def, nil
}

/*************************************************
* Helpers
**************************************************/
func getLabels(name string, component string) map[string]string {
	return map[string]string{"app.kubernetes.io/name": "ORDS",
		"app.kubernetes.io/instance":   name,
		"app.kubernetes.io/component":  component,
		"app.kubernetes.io/part-of":    "oracle-ords-operator",
		"app.kubernetes.io/created-by": "oracle-ords-controller-manager",
		"oracle.com/operator-filter":   "oracle-ords-operator",
	}
}

// func (r *RestDataServicesReconciler) getSecretValues(ctx context.Context, namespace string, secretName string, secretKey string) (string, error) {
// 	secretValue := &corev1.Secret{}
// 	if err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, secretValue); err != nil {
// 		return "", err
// 	}
// 	return string(secretValue.Data[secretKey]), nil
// }

func conditionalEntry(key string, value interface{}) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		if v != "" {
			return fmt.Sprintf(`  <entry key="%s">%s</entry>`+"\n", key, v)
		}
	case *int32:
		if v != nil {
			return fmt.Sprintf(`  <entry key="%s">%d</entry>`+"\n", key, *v)
		}
	case *bool:
		if v != nil {
			return fmt.Sprintf(`  <entry key="%s">%v</entry>`+"\n", key, *v)
		}
	case *time.Duration:
		if v != nil {
			return fmt.Sprintf(`  <entry key="%s">%v</entry>`+"\n", key, *v)
		}
	default:
		return fmt.Sprintf(`  <entry key="%s">%v</entry>`+"\n", key, v)
	}
	return ""
}
