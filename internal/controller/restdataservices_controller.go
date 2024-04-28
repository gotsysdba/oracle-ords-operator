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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
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
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	databasev1 "example.com/oracle-ords-operator/api/v1"
)

// Definitions of Standards
const (
	ordsConfigBase      = "/opt/oracle/sa/config"
	servicePortName     = "sa-svc-port"
	targetPortName      = "sa-pod-port"
	globalConfigMapName = "sa-settings-global"
	poolConfigPreName   = "sa-settings-" // Append PoolName
	controllerLabelKey  = "oracle.com/ords-operator-filter"
	controllerLabelVal  = "oracle-ords-operator"
	specHashLabel       = "oracle.com/ords-operator-spec-hash"
)

// Trigger a restart of Pods on Config Changes
var restartPods bool = false

// RestDataServicesReconciler reconciles a RestDataServices object
type RestDataServicesReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
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

// SetupWithManager sets up the controller with the Manager.
func (r *RestDataServicesReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&databasev1.RestDataServices{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&appsv1.Deployment{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&appsv1.DaemonSet{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

func (r *RestDataServicesReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logr := log.FromContext(ctx)
	ords := &databasev1.RestDataServices{}

	// Check if resource exists or was deleted
	if err := r.Get(ctx, req.NamespacedName, ords); err != nil {
		if apierrors.IsNotFound(err) {
			logr.Info("Resource deleted")
			return ctrl.Result{}, nil
		}
		logr.Error(err, "Error retrieving resource")
		return ctrl.Result{Requeue: true, RequeueAfter: time.Minute}, err
	}

	// ConfigMap - Global Settings
	if err := r.ConfigMapReconcile(ctx, ords, globalConfigMapName, 0); err != nil {
		logr.Error(err, "Error in ConfigMapReconcile (Global)")
		return ctrl.Result{}, err
	}

	// ConfigMap - Pool Settings
	definedPools := make(map[string]bool)
	for i := 0; i < len(ords.Spec.PoolSettings); i++ {
		poolConfigMapName := poolConfigPreName + strings.ToLower(ords.Spec.PoolSettings[i].PoolName)
		definedPools[poolConfigMapName] = true
		if err := r.ConfigMapReconcile(ctx, ords, poolConfigMapName, i); err != nil {
			logr.Error(err, "Error in ConfigMapReconcile (Pools)")
			return ctrl.Result{}, err
		}
	}
	if err := r.ConfigMapDelete(ctx, req, ords, definedPools); err != nil {
		logr.Error(err, "Error in ConfigMapDelete (Pools)")
		return ctrl.Result{}, err
	}

	// Workloads
	if err := r.WorkloadReconcile(ctx, ords, ords.Spec.WorkloadType); err != nil {
		logr.Error(err, "Error in WorkloadReconcile")
		return ctrl.Result{}, err
	}
	if err := r.WorkloadDelete(ctx, req, ords, ords.Spec.WorkloadType); err != nil {
		logr.Error(err, "Error in WorkloadDelete")
		return ctrl.Result{}, err
	}

	// Service
	if err := r.ServiceReconcile(ctx, ords); err != nil {
		logr.Error(err, "Error in WorkloadReconcile")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

/************************************************
 * ConfigMaps
 *************************************************/
func (r *RestDataServicesReconciler) ConfigMapReconcile(ctx context.Context, ords *databasev1.RestDataServices, configMapName string, poolIndex int) (err error) {
	logr := log.FromContext(ctx).WithName("ConfigMapReconcile")
	desiredConfigMap := r.ConfigMapDefine(ctx, ords, configMapName, 0)

	// Create if ConfigMap not found
	definedConfigMap := &corev1.ConfigMap{}
	if err = r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: ords.Namespace}, definedConfigMap); err != nil {
		if apierrors.IsNotFound(err) {
			if err := r.Create(ctx, desiredConfigMap); err != nil {
				return err
			}
			logr.Info("Created: " + configMapName)
			restartPods = ords.Spec.AutoRestart
			r.Recorder.Eventf(ords, corev1.EventTypeNormal, "Create", "ConfigMap %s Created", configMapName)
			// Requery for comparison
			r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: ords.Namespace}, definedConfigMap)
		} else {
			return err
		}
	}
	if !equality.Semantic.DeepEqual(definedConfigMap.Data, desiredConfigMap.Data) {
		if err = r.Update(ctx, desiredConfigMap); err != nil {
			return err
		}
		logr.Info("Updated: " + configMapName)
		restartPods = ords.Spec.AutoRestart
		r.Recorder.Eventf(ords, corev1.EventTypeNormal, "Update", "ConfigMap %s Updated", configMapName)
	}

	return nil
}

/************************************************
 * Workloads
 *************************************************/
func (r *RestDataServicesReconciler) WorkloadReconcile(ctx context.Context, ords *databasev1.RestDataServices, kind string) (err error) {
	logr := log.FromContext(ctx).WithName("WorkloadReconcile")
	objectMeta := objectMetaDefine(ords, ords.Name)
	selector := selectorDefine(ords)
	template := podTemplateSpecDefine(ords)

	var desiredWorkload client.Object
	var desiredSpecHash string
	var definedSpecHash string

	switch kind {
	case "StatefulSet":
		desiredWorkload = &appsv1.StatefulSet{
			ObjectMeta: objectMeta,
			Spec: appsv1.StatefulSetSpec{
				Replicas: &ords.Spec.Replicas,
				Selector: &selector,
				Template: template,
			},
		}
		desiredSpecHash = generateSpecHash(desiredWorkload.(*appsv1.StatefulSet).Spec)
		desiredWorkload.(*appsv1.StatefulSet).ObjectMeta.Labels[specHashLabel] = desiredSpecHash
	case "DaemonSet":
		desiredWorkload = &appsv1.DaemonSet{
			ObjectMeta: objectMeta,
			Spec: appsv1.DaemonSetSpec{
				Selector: &selector,
				Template: template,
			},
		}
		desiredSpecHash = generateSpecHash(desiredWorkload.(*appsv1.DaemonSet).Spec)
		desiredWorkload.(*appsv1.DaemonSet).ObjectMeta.Labels[specHashLabel] = desiredSpecHash
	default:
		desiredWorkload = &appsv1.Deployment{
			ObjectMeta: objectMeta,
			Spec: appsv1.DeploymentSpec{
				Replicas: &ords.Spec.Replicas,
				Selector: &selector,
				Template: template,
			},
		}
		desiredSpecHash = generateSpecHash(desiredWorkload.(*appsv1.Deployment).Spec)
		desiredWorkload.(*appsv1.Deployment).ObjectMeta.Labels[specHashLabel] = desiredSpecHash
	}

	if err := ctrl.SetControllerReference(ords, desiredWorkload, r.Scheme); err != nil {
		return err
	}

	definedWorkload := reflect.New(reflect.TypeOf(desiredWorkload).Elem()).Interface().(client.Object)
	if err = r.Get(ctx, types.NamespacedName{Name: ords.Name, Namespace: ords.Namespace}, definedWorkload); err != nil {
		if apierrors.IsNotFound(err) {
			if err := r.Create(ctx, desiredWorkload); err != nil {
				return err
			}
			logr.Info(fmt.Sprintf("Created: %s", kind))
			restartPods = false
			r.Recorder.Eventf(ords, corev1.EventTypeNormal, "Create", "Created %s", kind)
			// Requery for comparison
			r.Get(ctx, types.NamespacedName{Name: ords.Name, Namespace: ords.Namespace}, definedWorkload)
		} else {
			return err
		}
	}

	definedLabelsField := reflect.ValueOf(definedWorkload).Elem().FieldByName("ObjectMeta").FieldByName("Labels")
	if definedLabelsField.IsValid() {
		specHashValue := definedLabelsField.MapIndex(reflect.ValueOf(specHashLabel))
		if specHashValue.IsValid() {
			definedSpecHash = specHashValue.Interface().(string)
		} else {
			definedSpecHash = "undefined"
		}
	}

	if desiredSpecHash != definedSpecHash {
		if err := r.Client.Update(ctx, desiredWorkload); err != nil {
			return err
		}
		r.Recorder.Eventf(ords, corev1.EventTypeNormal, "Update", "Updated %s", kind)
	}
	if restartPods {
		logr.Info(fmt.Sprintf("Cycling: %s", kind))
		labelsField := reflect.ValueOf(desiredWorkload).Elem().FieldByName("Spec").FieldByName("Template").FieldByName("ObjectMeta").FieldByName("Labels")
		if labelsField.IsValid() {
			labels := labelsField.Interface().(map[string]string)
			labels["configMapChanged"] = time.Now().Format("20060102T150405Z")
			labelsField.Set(reflect.ValueOf(labels))
			if err := r.Update(ctx, desiredWorkload); err != nil {
				return err
			}
			r.Recorder.Eventf(ords, corev1.EventTypeNormal, "Restart", "Restarted %s", kind)
			restartPods = false
		}
	}
	return nil
}

// Service
func (r *RestDataServicesReconciler) ServiceReconcile(ctx context.Context, ords *databasev1.RestDataServices) (err error) {
	logr := log.FromContext(ctx).WithName("ServiceReconcile")

	port := int32(80)
	if ords.Spec.GlobalSettings.StandaloneHttpPort != nil {
		port = *ords.Spec.GlobalSettings.StandaloneHttpPort
	}
	desiredService := r.ServiceDefine(ctx, ords, port)

	definedService := &corev1.Service{}
	if err = r.Get(ctx, types.NamespacedName{Name: ords.Name, Namespace: ords.Namespace}, definedService); err != nil {
		if apierrors.IsNotFound(err) {
			if err := r.Create(ctx, desiredService); err != nil {
				return err
			}
			logr.Info("Created: Service")
			r.Recorder.Eventf(ords, corev1.EventTypeNormal, "Create", "Service %s Created", ords.Name)
			// Requery for comparison
			r.Get(ctx, types.NamespacedName{Name: ords.Name, Namespace: ords.Namespace}, definedService)
		} else {
			return err
		}
	}

	desiredServicePort := int32(80)
	if ords.Spec.GlobalSettings.StandaloneHttpPort != nil {
		desiredServicePort = *ords.Spec.GlobalSettings.StandaloneHttpPort
	}
	for _, existingPort := range definedService.Spec.Ports {
		if existingPort.Name == servicePortName {
			if existingPort.Port != desiredServicePort {
				if err := r.Update(ctx, desiredService); err != nil {
					return err
				}
				logr.Info("Updated Service Port: " + existingPort.Name)
				r.Recorder.Eventf(ords, corev1.EventTypeNormal, "Update", "Service Port %s Updated", existingPort.Name)
			}
		}
	}
	return nil
}

/*************************************************
 * Definers
 /************************************************/
// ConfigMap
func (r *RestDataServicesReconciler) ConfigMapDefine(ctx context.Context, ords *databasev1.RestDataServices, configMapName string, poolIndex int) *corev1.ConfigMap {
	defData := make(map[string]string)
	if configMapName == globalConfigMapName {
		// GlobalConfigMap
		defData = map[string]string{
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
		}
	} else {
		// PoolConfigMap
		defData = map[string]string{
			"pool.xml": fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
				`<!DOCTYPE properties SYSTEM "http://java.sun.com/dtd/properties.dtd">` + "\n" +
				`<properties>` + "\n" +
				`  <entry key="db.username">` + ords.Spec.PoolSettings[poolIndex].DbUsername + `</entry>` + "\n" +
				conditionalEntry("db.adminUser", ords.Spec.PoolSettings[poolIndex].DbAdminUser) +
				conditionalEntry("db.cdb.adminUser", ords.Spec.PoolSettings[poolIndex].DbCdbAdminUser) +
				conditionalEntry("apex.security.administrator.roles", ords.Spec.PoolSettings[poolIndex].ApexSecurityAdministratorRoles) +
				conditionalEntry("apex.security.user.roles", ords.Spec.PoolSettings[poolIndex].ApexSecurityUserRoles) +
				conditionalEntry("autoupgrade.api.aulocation", ords.Spec.PoolSettings[poolIndex].AutoupgradeApiAulocation) +
				conditionalEntry("autoupgrade.api.enabled", ords.Spec.PoolSettings[poolIndex].AutoupgradeApiEnabled) +
				conditionalEntry("autoupgrade.api.jvmlocation", ords.Spec.PoolSettings[poolIndex].AutoupgradeApiJvmlocation) +
				conditionalEntry("autoupgrade.api.loglocation", ords.Spec.PoolSettings[poolIndex].AutoupgradeApiLoglocation) +
				conditionalEntry("db.credentialsSource", ords.Spec.PoolSettings[poolIndex].DbCredentialsSource) +
				conditionalEntry("db.poolDestroyTimeout", ords.Spec.PoolSettings[poolIndex].DbPoolDestroyTimeout) +
				conditionalEntry("db.wallet.zip", ords.Spec.PoolSettings[poolIndex].DbWalletZip) +
				conditionalEntry("db.wallet.zip.path", ords.Spec.PoolSettings[poolIndex].DbWalletZipPath) +
				conditionalEntry("db.wallet.zip.service", ords.Spec.PoolSettings[poolIndex].DbWalletZipService) +
				conditionalEntry("debug.trackResources", ords.Spec.PoolSettings[poolIndex].DebugTrackResources) +
				conditionalEntry("feature.openservicebroker.exclude", ords.Spec.PoolSettings[poolIndex].FeatureOpenservicebrokerExclude) +
				conditionalEntry("feature.sdw", ords.Spec.PoolSettings[poolIndex].FeatureSdw) +
				conditionalEntry("http.cookie.filter", ords.Spec.PoolSettings[poolIndex].HttpCookieFilter) +
				conditionalEntry("jdbc.auth.admin.role", ords.Spec.PoolSettings[poolIndex].JdbcAuthAdminRole) +
				conditionalEntry("jdbc.cleanup.mode", ords.Spec.PoolSettings[poolIndex].JdbCleanupMode) +
				conditionalEntry("owa.trace.sql", ords.Spec.PoolSettings[poolIndex].OwaTraceSql) +
				conditionalEntry("plsql.gateway.mode", ords.Spec.PoolSettings[poolIndex].PlsqlGatewayMode) +
				conditionalEntry("security.jwt.profile.enabled", ords.Spec.PoolSettings[poolIndex].SecurityJwtProfileEnabled) +
				conditionalEntry("security.jwks.size", ords.Spec.PoolSettings[poolIndex].SecurityJwksSize) +
				conditionalEntry("security.jwks.connection.timeout", ords.Spec.PoolSettings[poolIndex].SecurityJwksConnectionTimeout) +
				conditionalEntry("security.jwks.read.timeout", ords.Spec.PoolSettings[poolIndex].SecurityJwksReadTimeout) +
				conditionalEntry("security.jwks.refresh.interval", ords.Spec.PoolSettings[poolIndex].SecurityJwksRefreshInterval) +
				conditionalEntry("security.jwt.allowed.skew", ords.Spec.PoolSettings[poolIndex].SecurityJwtAllowedSkew) +
				conditionalEntry("security.jwt.allowed.age", ords.Spec.PoolSettings[poolIndex].SecurityJwtAllowedAge) +
				conditionalEntry("security.jwt.allowed.age", ords.Spec.PoolSettings[poolIndex].SecurityValidationFunctionType) +
				conditionalEntry("db.connectionType", ords.Spec.PoolSettings[poolIndex].DbConnectionType) +
				conditionalEntry("db.customURL", ords.Spec.PoolSettings[poolIndex].DbCustomURL) +
				conditionalEntry("db.hostname", ords.Spec.PoolSettings[poolIndex].DbHostname) +
				conditionalEntry("db.port", ords.Spec.PoolSettings[poolIndex].DbPort) +
				conditionalEntry("db.servicename", ords.Spec.PoolSettings[poolIndex].DbServicename) +
				conditionalEntry("db.serviceNameSuffix", ords.Spec.PoolSettings[poolIndex].DbServiceNameSuffix) +
				conditionalEntry("db.sid", ords.Spec.PoolSettings[poolIndex].DbSid) +
				conditionalEntry("db.tnsAliasName", ords.Spec.PoolSettings[poolIndex].DbTnsAliasName) +
				conditionalEntry("db.tnsDirectory", ords.Spec.PoolSettings[poolIndex].DbTnsDirectory) +
				conditionalEntry("jdbc.DriverType", ords.Spec.PoolSettings[poolIndex].JdbcDriverType) +
				conditionalEntry("jdbc.InactivityTimeout", ords.Spec.PoolSettings[poolIndex].JdbcInactivityTimeout) +
				conditionalEntry("jdbc.InitialLimit", ords.Spec.PoolSettings[poolIndex].JdbcInitialLimit) +
				conditionalEntry("jdbc.MaxConnectionReuseCount", ords.Spec.PoolSettings[poolIndex].JdbcMaxConnectionReuseCount) +
				conditionalEntry("jdbc.MaxLimit", ords.Spec.PoolSettings[poolIndex].JdbcMaxLimit) +
				conditionalEntry("jdbc.auth.enabled", ords.Spec.PoolSettings[poolIndex].JdbcAuthEnabled) +
				conditionalEntry("jdbc.MaxStatementsLimit", ords.Spec.PoolSettings[poolIndex].JdbcMaxStatementsLimit) +
				conditionalEntry("jdbc.MinLimit", ords.Spec.PoolSettings[poolIndex].JdbcMinLimit) +
				conditionalEntry("jdbc.statementTimeout", ords.Spec.PoolSettings[poolIndex].JdbcStatementTimeout) +
				conditionalEntry("misc.defaultPage", ords.Spec.PoolSettings[poolIndex].MiscDefaultPage) +
				conditionalEntry("misc.pagination.maxRows", ords.Spec.PoolSettings[poolIndex].MiscPaginationMaxRows) +
				conditionalEntry("procedure.postProcess", ords.Spec.PoolSettings[poolIndex].ProcedurePostProcess) +
				conditionalEntry("procedure.preProcess", ords.Spec.PoolSettings[poolIndex].ProcedurePreProcess) +
				conditionalEntry("procedure.rest.preHook", ords.Spec.PoolSettings[poolIndex].ProcedureRestPreHook) +
				conditionalEntry("security.requestAuthenticationFunction", ords.Spec.PoolSettings[poolIndex].SecurityRequestAuthenticationFunction) +
				conditionalEntry("security.requestValidationFunction", ords.Spec.PoolSettings[poolIndex].SecurityRequestValidationFunction) +
				conditionalEntry("soda.defaultLimit", ords.Spec.PoolSettings[poolIndex].SodaDefaultLimit) +
				conditionalEntry("soda.maxLimit", ords.Spec.PoolSettings[poolIndex].SodaMaxLimit) +
				conditionalEntry("restEnabledSql.active", ords.Spec.PoolSettings[poolIndex].RestEnabledSqlActive) +
				`</properties>`),
		}
	}

	objectMeta := objectMetaDefine(ords, configMapName)
	def := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: objectMeta,
		Data:       defData,
	}

	// Set the ownerRef
	if err := ctrl.SetControllerReference(ords, def, r.Scheme); err != nil {
		return nil
	}
	return def
}

func objectMetaDefine(ords *databasev1.RestDataServices, name string) metav1.ObjectMeta {
	labels := getLabels(ords.Name)
	return metav1.ObjectMeta{
		Name:      name,
		Namespace: ords.Namespace,
		Labels:    labels,
	}
}

func selectorDefine(ords *databasev1.RestDataServices) metav1.LabelSelector {
	labels := getLabels(ords.Name)
	return metav1.LabelSelector{
		MatchLabels: labels,
	}
}

func podTemplateSpecDefine(ords *databasev1.RestDataServices) corev1.PodTemplateSpec {
	labels := getLabels(ords.Name)
	specVolumes, specVolumeMounts := VolumesDefine(ords)
	port := int32(8080)
	if ords.Spec.GlobalSettings.StandaloneHttpPort != nil {
		port = *ords.Spec.GlobalSettings.StandaloneHttpPort
	}

	podSpecTemplate :=
		corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: corev1.PodSpec{
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
				}}},
		}
	return podSpecTemplate
}

// Volumes
func VolumesDefine(ords *databasev1.RestDataServices) ([]corev1.Volume, []corev1.VolumeMount) {
	// Initialize the slice to hold specifications
	var volumes []corev1.Volume
	var volumeMounts []corev1.VolumeMount

	// Build volume specifications for globalSettings
	standaloneVolume := volumeBuild("standalone", "EmptyDir")
	globalWalletVolume := volumeBuild("sa-wallet-global", "EmptyDir")
	globalConfigVolume := volumeBuild(globalConfigMapName, "ConfigMap")
	volumes = append(volumes, standaloneVolume, globalWalletVolume, globalConfigVolume)

	standaloneVolumeMount := volumeMountBuild("standalone", ordsConfigBase+"/global/standalone/", false)
	globalWalletVolumeMount := volumeMountBuild("sa-wallet-global", ordsConfigBase+"/global/wallet/", false)
	globalConfigVolumeMount := volumeMountBuild(globalConfigMapName, ordsConfigBase+"/global/", false)
	volumeMounts = append(volumeMounts, standaloneVolumeMount, globalWalletVolumeMount, globalConfigVolumeMount)

	// Build volume specifications for each pool in poolSettings
	for i := 0; i < len(ords.Spec.PoolSettings); i++ {
		poolName := strings.ToLower(ords.Spec.PoolSettings[i].PoolName)
		poolConfigName := poolConfigPreName + poolName
		poolWalletName := "sa-wallet-" + poolName
		// Volumes
		poolWalletVolume := volumeBuild(poolWalletName, "EmptyDir")
		poolConfigVolume := volumeBuild(poolConfigName, "ConfigMap")
		volumes = append(volumes, poolWalletVolume, poolConfigVolume)

		// VolumeMounts
		poolWalletVolumeMount := volumeMountBuild(poolWalletName, ordsConfigBase+"/databases/"+poolName+"/wallet", false)
		poolConfigVolumeMount := volumeMountBuild(poolConfigName, ordsConfigBase+"/databases/"+poolName+"/", false)
		volumeMounts = append(volumeMounts, poolWalletVolumeMount, poolConfigVolumeMount)
	}
	return volumes, volumeMounts
}

func volumeMountBuild(name string, path string, readOnly bool) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      name,
		MountPath: path,
		ReadOnly:  readOnly,
	}
}

func volumeBuild(name string, source string) corev1.Volume {
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
	case "Secret":
		return corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: name,
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
func (r *RestDataServicesReconciler) ServiceDefine(ctx context.Context, ords *databasev1.RestDataServices, port int32) *corev1.Service {
	labels := getLabels(ords.Name)

	objectMeta := objectMetaDefine(ords, ords.Name)
	def := &corev1.Service{
		ObjectMeta: objectMeta,
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

	// Set the ownerRef
	if err := ctrl.SetControllerReference(ords, def, r.Scheme); err != nil {
		return nil
	}
	return def
}

/*************************************************
 * Deletions
 **************************************************/
func (r *RestDataServicesReconciler) ConfigMapDelete(ctx context.Context, req ctrl.Request, ords *databasev1.RestDataServices, definedPools map[string]bool) (err error) {
	// Delete Undefined Pool ConfigMaps
	configMapList := &corev1.ConfigMapList{}
	if err := r.List(ctx, configMapList, client.InNamespace(req.Namespace),
		client.MatchingLabels(map[string]string{controllerLabelKey: controllerLabelVal}),
	); err != nil {
		return err
	}

	for _, configMap := range configMapList.Items {
		if configMap.Name == globalConfigMapName {
			continue
		}
		if _, exists := definedPools[configMap.Name]; !exists {
			if err := r.Delete(ctx, &configMap); err != nil {
				return err
			}
			restartPods = ords.Spec.AutoRestart
			r.Recorder.Eventf(ords, corev1.EventTypeNormal, "Delete", "ConfigMap %s Deleted", configMap.Name)
		}
	}
	return nil
}

func (r *RestDataServicesReconciler) WorkloadDelete(ctx context.Context, req ctrl.Request, ords *databasev1.RestDataServices, kind string) (err error) {
	logr := log.FromContext(ctx).WithName("WorkloadDelete")

	// Get Workloads
	deploymentList := &appsv1.DeploymentList{}
	if err := r.List(ctx, deploymentList, client.InNamespace(req.Namespace),
		client.MatchingLabels(map[string]string{controllerLabelKey: controllerLabelVal}),
	); err != nil {
		return err
	}

	statefulSetList := &appsv1.StatefulSetList{}
	if err := r.List(ctx, statefulSetList, client.InNamespace(req.Namespace),
		client.MatchingLabels(map[string]string{controllerLabelKey: controllerLabelVal}),
	); err != nil {
		return err
	}

	daemonSetList := &appsv1.DaemonSetList{}
	if err := r.List(ctx, daemonSetList, client.InNamespace(req.Namespace),
		client.MatchingLabels(map[string]string{controllerLabelKey: controllerLabelVal}),
	); err != nil {
		return err
	}

	switch kind {
	case "StatefulSet":
		for _, deleteDaemonSet := range daemonSetList.Items {
			if err := r.Delete(ctx, &deleteDaemonSet); err != nil {
				return err
			}
			logr.Info("Deleted: " + kind)
			r.Recorder.Eventf(ords, corev1.EventTypeNormal, "Delete", "Workload %s Deleted", kind)
		}
		for _, deleteDeployment := range deploymentList.Items {
			if err := r.Delete(ctx, &deleteDeployment); err != nil {
				return err
			}
			logr.Info("Deleted: " + kind)
			r.Recorder.Eventf(ords, corev1.EventTypeNormal, "Delete", "Workload %s Deleted", kind)
		}
	case "DaemonSet":
		for _, deleteDeployment := range deploymentList.Items {
			if err := r.Delete(ctx, &deleteDeployment); err != nil {
				return err
			}
			logr.Info("Deleted: " + kind)
			r.Recorder.Eventf(ords, corev1.EventTypeNormal, "Delete", "Workload %s Deleted", kind)
		}
		for _, deleteStatefulSet := range statefulSetList.Items {
			if err := r.Delete(ctx, &deleteStatefulSet); err != nil {
				return err
			}
			logr.Info("Deleted StatefulSet: " + deleteStatefulSet.Name)
			r.Recorder.Eventf(ords, corev1.EventTypeNormal, "Delete", "Workload %s Deleted", kind)
		}
	default:
		for _, deleteStatefulSet := range statefulSetList.Items {
			if err := r.Delete(ctx, &deleteStatefulSet); err != nil {
				return err
			}
			logr.Info("Deleted: " + kind)
			r.Recorder.Eventf(ords, corev1.EventTypeNormal, "Delete", "Workload %s Deleted", kind)
		}
		for _, deleteDaemonSet := range daemonSetList.Items {
			if err := r.Delete(ctx, &deleteDaemonSet); err != nil {
				return err
			}
			logr.Info("Deleted: " + kind)
			r.Recorder.Eventf(ords, corev1.EventTypeNormal, "Delete", "Workload %s Deleted", kind)
		}
	}
	return nil
}

/*************************************************
 * Helpers
 **************************************************/
func getLabels(name string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/instance": name,
		controllerLabelKey:           controllerLabelVal,
	}
}

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

func generateSpecHash(spec interface{}) string {
	byteArray, err := json.Marshal(spec)
	if err != nil {
		return ""
	}

	hash := sha256.New()
	_, err = hash.Write(byteArray)
	if err != nil {
		return ""
	}

	hashBytes := hash.Sum(nil)
	hashString := hex.EncodeToString(hashBytes[:8])

	return hashString
}
