package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

func (r *RestDataServicesReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	ords := &databasev1.RestDataServices{}

	// Check if there is an ORDS resource; if not nothing to reconcile
	if err := r.Get(ctx, req.NamespacedName, ords); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Unable to retrieve an RestDataServices CRD.", "connection", ords)
		return ctrl.Result{}, err
	}

	// Set the status as Unknown when no status are available
	if ords.Status.Conditions == nil || len(ords.Status.Conditions) == 0 {
		meta.SetStatusCondition(&ords.Status.Conditions, metav1.Condition{Type: typeAvailable, Status: metav1.ConditionUnknown, Reason: "Reconciling", Message: "Starting reconciliation"})
		if err := r.Status().Update(ctx, ords); err != nil {
			logger.Error(err, "Failed to update", "obj", ords)
			return ctrl.Result{}, err
		}
		if err := r.Get(ctx, req.NamespacedName, ords); err != nil {
			logger.Error(err, "Failed to re-fetch", "obj", ords)
			return ctrl.Result{}, err
		}
	}

	// Check if the workload already exists, if not create a new one
	found := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: ords.Name, Namespace: ords.Namespace}, found)
	if err != nil && apierrors.IsNotFound(err) {
		// Define a new deployment
		dep, err := r.deploymentForRestDataServices(ords)
		if err != nil {
			logger.Error(err, "Failed to define new Deployment resource for RestDataServices")

			// The following implementation will update the status
			meta.SetStatusCondition(&ords.Status.Conditions, metav1.Condition{Type: typeAvailable,
				Status: metav1.ConditionFalse, Reason: "Reconciling",
				Message: fmt.Sprintf("Failed to create Deployment for the custom resource (%s): (%s)", ords.Name, err)})

			if err := r.Status().Update(ctx, ords); err != nil {
				logger.Error(err, "Failed to update RestDataServices status")
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, err
		}

		logger.Info("Creating a new Deployment",
			"Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		if err = r.Create(ctx, dep); err != nil {
			logger.Error(err, "Failed to create new Deployment",
				"Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}

		// Deployment created successfully
		// We will requeue the reconciliation so that we can ensure the state
		// and move forward for the next operations
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	} else if err != nil {
		logger.Error(err, "Failed to get Deployment")
		// Let's return the error for the reconciliation be re-trigged again
		return ctrl.Result{}, err
	}

	// Reconcile Replicas
	replicas := ords.Spec.Replicas
	if *found.Spec.Replicas != replicas {
		logger.Info("Scaling", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
		found.Spec.Replicas = &replicas
		if err = r.Update(ctx, found); err != nil {
			logger.Error(err, "Failed to update Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)

			if err := r.Get(ctx, req.NamespacedName, ords); err != nil {
				logger.Error(err, "Failed to re-fetch ords")
				return ctrl.Result{}, err
			}

			// The following implementation will update the status
			meta.SetStatusCondition(&ords.Status.Conditions, metav1.Condition{Type: typeAvailable,
				Status: metav1.ConditionFalse, Reason: "Resizing",
				Message: fmt.Sprintf("Failed to update the size for the custom resource (%s): (%s)", ords.Name, err)})

			if err := r.Status().Update(ctx, ords); err != nil {
				logger.Error(err, "Failed to update ords status")
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	meta.SetStatusCondition(&ords.Status.Conditions, metav1.Condition{Type: typeAvailable,
		Status: metav1.ConditionTrue, Reason: "Reconciling",
		Message: fmt.Sprintf("Deployment for custom resource (%s) with %d replicas created successfully", ords.Name, replicas)})

	if err := r.Status().Update(ctx, ords); err != nil {
		logger.Error(err, "Failed to update status", "obj", ords)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// deployment
func (r *RestDataServicesReconciler) deploymentForRestDataServices(ords *databasev1.RestDataServices) (*appsv1.Deployment, error) {
	ls := labelsForRestDataServices(ords.Name, ords.Spec.Image)
	podTemplate := podForRestDataServices(ords)
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

	// Set the ownerRef for the Deployment
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
	if err := ctrl.SetControllerReference(ords, dep, r.Scheme); err != nil {
		return nil, err
	}
	return dep, nil
}

func podForRestDataServices(ords *databasev1.RestDataServices) corev1.PodSpec {
	podTemplate := corev1.PodSpec{
		SecurityContext: &corev1.PodSecurityContext{
			RunAsNonRoot: &[]bool{true}[0],
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeRuntimeDefault,
			},
		},
		Containers: []corev1.Container{{
			Image:           ords.Spec.Image,
			Name:            fmt.Sprintf("%s-ords", ords.Name),
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
				ContainerPort: ords.Spec.Port,
				Name:          "standalone-port",
			}},
			Command: []string{"ords", "serve"},
		}},
	}
	return podTemplate
}

func labelsForRestDataServices(name string, image string) map[string]string {
	var imageTag string
	imageTag = strings.Split(image, ":")[1]

	return map[string]string{"app.kubernetes.io/name": "ORDS",
		"app.kubernetes.io/instance":   name,
		"app.kubernetes.io/version":    imageTag,
		"app.kubernetes.io/part-of":    "oracle-ords-operator",
		"app.kubernetes.io/created-by": "controller-manager",
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *RestDataServicesReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&databasev1.RestDataServices{}).
		Complete(r)
}
