/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RestDataServicesSpec defines the desired state of RestDataServices
type RestDataServicesSpec struct {
	Image              OracleRestDataServiceImage               `json:"image,omitempty"`

	// +k8s:openapi-gen=true
	// +kubebuilder:validation:Minimum=1
	Replicas int32 `json:"replicas,omitempty"`
}

// OracleRestDataServiceImage defines the Image source and pullSecrets for POD
type OracleRestDataServiceImage struct {
	Version     string `json:"version,omitempty"`
	PullFrom    string `json:"pullFrom"`
	PullSecrets string `json:"pullSecrets,omitempty"`
}

// RestDataServicesStatus defines the observed state of RestDataServices
type RestDataServicesStatus struct {
	Image OracleRestDataServiceImage `json:"image,omitempty"`
	Replicas int32 `json:"replicas,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// RestDataServices is the Schema for the restdataservices API
type RestDataServices struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RestDataServicesSpec   `json:"spec,omitempty"`
	Status RestDataServicesStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RestDataServicesList contains a list of RestDataServices
type RestDataServicesList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RestDataServices `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RestDataServices{}, &RestDataServicesList{})
}
