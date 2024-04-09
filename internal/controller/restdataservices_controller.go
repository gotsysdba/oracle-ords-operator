package controller

import (
	"context"
	"fmt"

	//"strings"
	"time"

	//appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	databasev1 "github.com/gotsysdba/oracle-ords-operator/api/v1"
)

const RestDataServicesFinalizer = "database.oracle.com/restdataservicesfinalizer"

// Definitions to manage status conditions
const (
	// typeAvailable represents the status of the Deployment reconciliation
	typeAvailable = "Available"
	// typeDegraded represents the status deleted and the finalizer operations are must to occur.
	typeDegraded = "Degraded"
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
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

func (r *RestDataServicesReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logr := log.FromContext(ctx)
	ords := &databasev1.RestDataServices{}

	// Check if there is an ORDS resource; if not nothing to reconcile
	if err := r.Get(ctx, req.NamespacedName, ords); err != nil {
		if apierrors.IsNotFound(err) {
			logr.Info("No RestDataServices resources found")
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

	// Default ConfigMap
	existingConfigMap := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: "ords-default-config", Namespace: ords.Namespace}, existingConfigMap)
	if err != nil && apierrors.IsNotFound(err) {
		logr.Info("Missing Default ConfigMap, Creating")
		def, err := r.defConfigMap(ctx, ords)
		if err != nil {
			logr.Error(err, "Failed to define new ConfigMap for RestDataServices")
			condition := metav1.Condition{
				Type: typeAvailable, Status: metav1.ConditionFalse,
				Reason: "RequirementsNotMet", Message: "Default ConfigMap does not exist",
			}
			err := r.updateStatus(ctx, req, ords, condition)
			return ctrl.Result{}, err
		}
		logr.Info("Creating ConfigMap", "Namespace", def.Namespace, "Name", def.Name)
		if err = r.Create(ctx, def); err != nil {
			logr.Error(err, "Failed creating new ConfigMap", "Namespace", def.Namespace, "Name", def.Name)
			return ctrl.Result{}, err
		}
	} else {
		logr.Info("Found Default ConfigMap, Reconciling")
		newConfigMap, err := r.defConfigMap(ctx, ords)
		if err != nil {
			logr.Error(err, "Failed to define comparable ConfigMap for RestDataServices")
			condition := metav1.Condition{
				Type: typeAvailable, Status: metav1.ConditionFalse,
				Reason: "ResourceFound", Message: "Starting ConfigMap Reconciliation",
			}
			err := r.updateStatus(ctx, req, ords, condition)
			return ctrl.Result{}, err
		}
		if equality.Semantic.DeepEqual(existingConfigMap.Data, newConfigMap.Data) {
			logr.Info("ConfigMaps are the same. No action needed.")
			return ctrl.Result{}, nil
		}
		logr.Info("Updating ConfigMap", "Namespace", newConfigMap.Namespace, "Name", newConfigMap.Name)
		if err := r.Update(ctx, newConfigMap); err != nil {
			logr.Error(err, "Failed updating ConfigMap", "Namespace", newConfigMap.Namespace, "Name", newConfigMap.Name)
			return ctrl.Result{}, err
		}
	}

	// Set CR Status
	condition := metav1.Condition{Type: typeAvailable, Status: metav1.ConditionTrue,
		Reason: "Succeeded", Message: fmt.Sprintf("Resource (%s) created successfully", ords.Name)}
	err = r.updateStatus(ctx, req, ords, condition)
	return ctrl.Result{}, err
}

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

// ConfigMaps
func (r *RestDataServicesReconciler) defConfigMap(ctx context.Context, ords *databasev1.RestDataServices) (*corev1.ConfigMap, error) {
	ls := labelsForRestDataServices(ords.Name)
	def := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ords-default-config",
			Namespace: ords.Namespace,
			Labels:    ls,
		},
		Data: map[string]string{
			"settings.xml": `<?xml version="1.0" encoding="UTF-8"?>
							<!DOCTYPE properties SYSTEM "http://java.sun.com/dtd/properties.dtd">
							<properties>
							<comment>Default created by Controller</comment>
							<entry key="database.api.enabled">false</entry>
							<entry key="debug.printDebugToScreen">false</entry>
							<entry key="feature.sdw">false</entry>
							<entry key="jdbc.InitialLimit">10</entry>
							<entry key="jdbc.MaxLimit">100</entry>
							<entry key="logr.procedure">false</entry>
							<entry key="misc.defaultPage">apex</entry>
							<entry key="restEnabledSql.active">false</entry>
							<entry key="security.httpsHeaderCheck">X-Forwarded-Proto: https</entry>
							<entry key="standalone.context.path">/</entry>
							<entry key="standalone.http.port">8080</entry>
							<entry key="standalone.static.context.path">/i</entry>
							</properties>`,
		},
	}

	// Set the ownerRef
	if err := ctrl.SetControllerReference(ords, def, r.Scheme); err != nil {
		return nil, err
	}
	return def, nil
}

func labelsForRestDataServices(name string) map[string]string {
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
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
