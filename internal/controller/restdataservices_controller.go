package controller

import (
	"context"
	"fmt"
	"strings"

	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	databasev1 "github.com/gotsysdba/oracle-ords-operator/api/v1"
)

const RestDataServicesFinalizer = "database.oracle.com/restdataservicesfinalizer"

// Definitions to manage status conditions
const (
	typeAvailable = "Available"
	typeDegraded  = "Degraded"
)

// Definitions of Standards
const (
	ordsConfigBase    = "/opt/oracle/standalone/config"
	servicePortName   = "sa-svc-port"
	targetPortName    = "sa-pod-port"
	globalConfigName  = "sa-settings-global"
	poolConfigPreName = "sa-settings-" // Append PoolName
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
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=deployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

func (r *RestDataServicesReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logr := log.FromContext(ctx)
	ords := &databasev1.RestDataServices{}

	// Check if there is an ORDS resource; if not nothing to reconcile
	if err := r.Get(ctx, req.NamespacedName, ords); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		// The CR is not defined... something has gone horribly wrong!!
		logr.Error(err, "Unable to retrieve an RestDataServices CRD.", "connection", ords)
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	// Set the status as Unknown when no status are available
	if ords.Status.Conditions == nil || len(ords.Status.Conditions) == 0 {
		condition := metav1.Condition{
			Type: typeAvailable, Status: metav1.ConditionUnknown,
			Reason: "Reconciling", Message: "Starting reconciliation",
		}
		err := r.updateStatus(ctx, req, ords, condition)
		return ctrl.Result{}, err
	}

	/*************************************************
	* Global ConfigMap
	/************************************************/
	existingConfigMap := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: globalConfigName, Namespace: ords.Namespace}, existingConfigMap)
	if err != nil && apierrors.IsNotFound(err) {
		logr.Info("Missing Global ConfigMap, Creating")
		def, err := r.defGlobalConfigMap(ctx, ords)
		if err != nil {
			logr.Error(err, "Failed to define new ConfigMap for RestDataServices")
			condition := metav1.Condition{
				Type: typeAvailable, Status: metav1.ConditionFalse,
				Reason: "RequirementsNotMet", Message: "Global ConfigMap does not exist",
			}
			err := r.updateStatus(ctx, req, ords, condition)
			return ctrl.Result{}, err
		}
		if err = r.Create(ctx, def); err != nil {
			logr.Error(err, "Failed creating new ConfigMap", "Namespace", def.Namespace, "Name", def.Name)
			return ctrl.Result{}, err
		}
		logr.Info("Created ConfigMap", "Namespace", def.Namespace, "Name", def.Name)
	} else {
		logr.Info("Found Global ConfigMap, Reconciling")
		newConfigMap, err := r.defGlobalConfigMap(ctx, ords)
		if err != nil {
			logr.Error(err, "Failed to define comparable ConfigMap for RestDataServices")
			condition := metav1.Condition{
				Type: typeAvailable, Status: metav1.ConditionFalse,
				Reason: "ResourceFound", Message: "Starting ConfigMap Reconciliation",
			}
			err := r.updateStatus(ctx, req, ords, condition)
			return ctrl.Result{}, err
		}
		if !equality.Semantic.DeepEqual(existingConfigMap.Data, newConfigMap.Data) {
			if err := r.Update(ctx, newConfigMap); err != nil {
				logr.Error(err, "Failed updating ConfigMap", "Namespace", newConfigMap.Namespace, "Name", newConfigMap.Name)
				return ctrl.Result{}, err
			}
			logr.Info("Updated ConfigMap", "Namespace", newConfigMap.Namespace, "Name", newConfigMap.Name)
			// Update deployment's pod label to trigger pod restart
			deployment := &appsv1.Deployment{}
			err = r.Get(ctx, types.NamespacedName{Name: ords.Name, Namespace: ords.Namespace}, deployment)
			if err == nil {
				deployment.Spec.Template.ObjectMeta.Labels["configMapChanged"] = time.Now().Format("20060102T150405Z")
				if err := r.Update(ctx, deployment); err != nil {
					return ctrl.Result{}, err
				}
			}
		}
	}

	/*************************************************
	* Pool ConfigMap
	/************************************************/
	for i := 0; i < len(ords.Spec.PoolSettings); i++ {
		poolName := ords.Spec.PoolSettings[i].PoolName
		poolConfigMapName := poolConfigPreName + strings.ToLower(poolName)
		err = r.Get(ctx, types.NamespacedName{Name: poolConfigMapName, Namespace: ords.Namespace}, existingConfigMap)
		if err != nil && apierrors.IsNotFound(err) {
			logr.Info("Missing Pool ConfigMap, Creating")
			def, err := r.defPoolConfigMap(ctx, ords, poolConfigMapName, i)
			if err != nil {
				logr.Error(err, "Failed to define new ConfigMap for RestDataServices")
				condition := metav1.Condition{
					Type: typeAvailable, Status: metav1.ConditionFalse,
					Reason: "RequirementsNotMet", Message: "Global ConfigMap does not exist",
				}
				err := r.updateStatus(ctx, req, ords, condition)
				return ctrl.Result{}, err
			}
			if err = r.Create(ctx, def); err != nil {
				logr.Error(err, "Failed creating new ConfigMap", "Namespace", def.Namespace, "Name", def.Name)
				return ctrl.Result{}, err
			}
			logr.Info("Created ConfigMap", "Namespace", def.Namespace, "Name", def.Name)
		} else {
			logr.Info("Found Pool ConfigMap, Reconciling")
			newConfigMap, err := r.defPoolConfigMap(ctx, ords, poolConfigMapName, i)
			if err != nil {
				logr.Error(err, "Failed to define comparable ConfigMap for RestDataServices")
				condition := metav1.Condition{
					Type: typeAvailable, Status: metav1.ConditionFalse,
					Reason: "ResourceFound", Message: "Starting ConfigMap Reconciliation",
				}
				err := r.updateStatus(ctx, req, ords, condition)
				return ctrl.Result{}, err
			}
			if !equality.Semantic.DeepEqual(existingConfigMap.Data, newConfigMap.Data) {
				if err := r.Update(ctx, newConfigMap); err != nil {
					logr.Error(err, "Failed updating ConfigMap", "Namespace", newConfigMap.Namespace, "Name", newConfigMap.Name)
					return ctrl.Result{}, err
				}
				logr.Info("Updated ConfigMap", "Namespace", newConfigMap.Namespace, "Name", newConfigMap.Name)
				// Update deployment's pod label to trigger pod restart
				deployment := &appsv1.Deployment{}
				err = r.Get(ctx, types.NamespacedName{Name: ords.Name, Namespace: ords.Namespace}, deployment)
				if err == nil {
					deployment.Spec.Template.ObjectMeta.Labels["configMapChanged"] = time.Now().Format("20060102T150405Z")
					if err := r.Update(ctx, deployment); err != nil {
						return ctrl.Result{}, err
					}
				}
			}
		}
	}

	/*************************************************
	* Deployment
	/************************************************/
	existingDeployment := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: ords.Name, Namespace: ords.Namespace}, existingDeployment)
	if err != nil && apierrors.IsNotFound(err) {
		logr.Info("Missing Deployment, Creating")
		def, err := r.defDeployment(ctx, ords)
		if err != nil {
			logr.Error(err, "Failed to define new Deployment for RestDataServices")
			condition := metav1.Condition{
				Type: typeAvailable, Status: metav1.ConditionFalse,
				Reason: "RequirementsNotMet", Message: "Deployment does not exist",
			}
			err := r.updateStatus(ctx, req, ords, condition)
			return ctrl.Result{}, err
		}
		if err = r.Create(ctx, def); err != nil {
			logr.Error(err, "Failed creating new Deployment", "Namespace", def.Namespace, "Name", def.Name)
			return ctrl.Result{}, err
		}
		logr.Info("Created Deployment", "Namespace", def.Namespace, "Name", def.Name)
	} else {
		definedReplicas := ords.Spec.Replicas
		if *existingDeployment.Spec.Replicas != definedReplicas {
			logr.Info("Scaling Deployment", "Deployment.Namespace", existingDeployment.Namespace, "Deployment.Name", existingDeployment.Name)
			existingDeployment.Spec.Replicas = &definedReplicas
			if err := r.Update(ctx, existingDeployment); err != nil {
				logr.Error(err, "Failed scaling Deployment", "Namespace", existingDeployment.Namespace, "Name", existingDeployment.Name)
				return ctrl.Result{}, err
			}
		}
	}

	/*************************************************
	* Service
	/************************************************/
	existingService := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: ords.Name, Namespace: ords.Namespace}, existingService)
	if err != nil && apierrors.IsNotFound(err) {
		logr.Info("Missing Service, Creating")
		def, err := r.defService(ctx, ords)
		if err != nil {
			logr.Error(err, "Failed to define new Service for RestDataServices")
			condition := metav1.Condition{
				Type: typeAvailable, Status: metav1.ConditionFalse,
				Reason: "RequirementsNotMet", Message: "Service does not exist",
			}
			err := r.updateStatus(ctx, req, ords, condition)
			return ctrl.Result{}, err
		}
		if err = r.Create(ctx, def); err != nil {
			logr.Error(err, "Failed creating new Service", "Namespace", def.Namespace, "Name", def.Name)
			return ctrl.Result{}, err
		}
		logr.Info("Created Service", "Namespace", def.Namespace, "Name", def.Name)
	} else {
		definedServicePort := ords.Spec.ServicePort
		for _, existingPort := range existingService.Spec.Ports {
			if existingPort.Name == servicePortName {
				if existingPort.Port != definedServicePort {
					existingPort.Port = definedServicePort
					if err := r.Update(ctx, existingService); err != nil {
						logr.Error(err, "Failed reconciling ServicePort", "Namespace", existingService.Namespace, "Name", existingService.Name)
						return ctrl.Result{}, err
					}
				}
			}
		}
	}

	// Set CR Status
	condition := metav1.Condition{Type: typeAvailable, Status: metav1.ConditionTrue,
		Reason: "Succeeded", Message: fmt.Sprintf("Resource (%s) created successfully", ords.Name)}
	err = r.updateStatus(ctx, req, ords, condition)
	return ctrl.Result{}, err
}

/*************************************************
* Helpers
/************************************************/
// UpdateStatus
func (r *RestDataServicesReconciler) updateStatus(ctx context.Context, req ctrl.Request, ords *databasev1.RestDataServices, condition metav1.Condition) error {
	logr := log.FromContext(ctx).WithName("updateStatus")
	meta.SetStatusCondition(&ords.Status.Conditions, condition)
	if err := r.Status().Update(ctx, ords); err != nil {
		logr.Error(err, "Failed to update RestDataServices status")
		return err
	}
	if err := r.Get(ctx, req.NamespacedName, ords); err != nil {
		logr.Error(err, "Failed to re-fetch RestDataServices")
		return err
	}
	return nil
}

// Global ConfigMap
func (r *RestDataServicesReconciler) defGlobalConfigMap(ctx context.Context, ords *databasev1.RestDataServices) (*corev1.ConfigMap, error) {
	ls := getLabels(ords.Name)
	def := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      globalConfigName,
			Namespace: ords.Namespace,
			Labels:    ls,
		},
		Data: map[string]string{
			"settings.xml": fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
				`<!DOCTYPE properties SYSTEM "http://java.sun.com/dtd/properties.dtd">` + "\n" +
				`<properties>` + "\n" +
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
				conditionalEntry("standalone.https.cert", ords.Spec.GlobalSettings.StandaloneHttpsCert) +
				conditionalEntry("standalone.https.cert.key", ords.Spec.GlobalSettings.StandaloneHttpsCertKey) +
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
	ls := getLabels(ords.Name)
	def := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      poolConfigName,
			Namespace: ords.Namespace,
			Labels:    ls,
		},
		Data: map[string]string{
			"pool.xml": fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
				`<!DOCTYPE properties SYSTEM "http://java.sun.com/dtd/properties.dtd">` + "\n" +
				`<properties>` + "\n" +
				conditionalEntry("apex.security.administrator.roles", ords.Spec.PoolSettings[i].ApexSecurityAdministratorRoles) +
				conditionalEntry("apex.security.user.roles", ords.Spec.PoolSettings[i].ApexSecurityUserRoles) +
				conditionalEntry("autoupgrade.api.aulocation", ords.Spec.PoolSettings[i].AutoupgradeApiAulocation) +
				conditionalEntry("autoupgrade.api.enabled", ords.Spec.PoolSettings[i].AutoupgradeApiEnabled) +
				conditionalEntry("autoupgrade.api.jvmlocation", ords.Spec.PoolSettings[i].AutoupgradeApiJvmlocation) +
				conditionalEntry("autoupgrade.api.loglocation", ords.Spec.PoolSettings[i].AutoupgradeApiLoglocation) +
				conditionalEntry("db.adminUser", ords.Spec.PoolSettings[i].DbAdminUser) +
				conditionalEntry("db.cdb.adminUser", ords.Spec.PoolSettings[i].DbCdbAdminUser) +
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
				conditionalEntry("db.username", ords.Spec.PoolSettings[i].DbUsername) +
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

// Deployments
func (r *RestDataServicesReconciler) defDeployment(ctx context.Context, ords *databasev1.RestDataServices) (*appsv1.Deployment, error) {
	ls := getLabels(ords.Name)
	podTemplate := defPods(ords)
	replicas := ords.Spec.Replicas

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ords.Name,
			Namespace: ords.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: podTemplate,
			},
		},
	}

	if err := ctrl.SetControllerReference(ords, dep, r.Scheme); err != nil {
		return nil, err
	}
	return dep, nil
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
			Command: []string{"/bin/bash", "-c", "ords --config $ORDS_CONFIG serve"},
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
	globalVolume := buildVolume(globalConfigName)
	volumes = append(volumes, globalVolume)

	globalVolumeMount := buildVolumeMount(globalConfigName, ordsConfigBase+"/global/")
	volumeMounts = append(volumeMounts, globalVolumeMount)

	// Build volume specifications for each pool in poolSettings
	for _, pool := range ords.Spec.PoolSettings {
		poolName := strings.ToLower(pool.PoolName)
		poolConfigName := poolConfigPreName + poolName
		poolVolume := buildVolume(poolConfigName)
		volumes = append(volumes, poolVolume)

		poolVolumeMount := buildVolumeMount(poolConfigName, ordsConfigBase+"/database/"+poolName+"/")
		volumeMounts = append(volumeMounts, poolVolumeMount)

	}
	return volumes, volumeMounts
}

func buildVolumeMount(name string, path string) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      name,
		MountPath: path,
		ReadOnly:  false,
	}
}

func buildVolume(name string) corev1.Volume {
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
}

// Service
func (r *RestDataServicesReconciler) defService(ctx context.Context, ords *databasev1.RestDataServices) (*corev1.Service, error) {
	port := int32(80)
	if ords.Spec.GlobalSettings.StandaloneHttpPort != nil {
		port = *ords.Spec.GlobalSettings.StandaloneHttpPort
	}
	ls := getLabels(ords.Name)
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ords.Name,
			Namespace: ords.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: ls,
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
	if err := ctrl.SetControllerReference(ords, svc, r.Scheme); err != nil {
		return nil, err
	}
	return svc, nil
}

// Helpers
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

func getLabels(name string) map[string]string {
	return map[string]string{"app.kubernetes.io/name": "ORDS",
		"app.kubernetes.io/instance":   name,
		"app.kubernetes.io/part-of":    "oracle-ords-operator",
		"app.kubernetes.io/created-by": "controller-manager",
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *RestDataServicesReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&databasev1.RestDataServices{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
