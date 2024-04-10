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
	"net"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RestDataServicesSpec defines the desired state of RestDataServices
type RestDataServicesSpec struct {
	WorkloadType string `json:"workloadType,omitempty"`
	// +kubebuilder:validation:Minimum=1
	Replicas         int32             `json:"replicas,omitempty"`
	Image            string            `json:"image" protobuf:"bytes,2,opt,name=image"`
	ImagePullPolicy  corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	ImagePullSecrets string            `json:"imagePullSecrets,omitempty"`
	// Contains settings that are configured across the entire ORDS instance.
	GlobalSettings GlobalSettings `json:"globalSettings"`
	// +k8s:openapi-gen=true

}

type GlobalSettings struct {
	// Specifies the duration after a GraphQL schema is not accessed from the cache that it expires.
	// +kubebuilder:default=8h
	CacheMetadataGraphqlExpireAfterAccess time.Duration `json:"cacheMetadataGraphqlExpireAfterAccess,omitempty" default:"8h"`

	// Specifies the setting to enable or disable JWKS caching.
	// +kubebuilder:default=true
	CacheMetadataJwksEnabled bool `json:"cacheMetadataJwksEnabled,omitempty" default:"true"`

	// Specifies the initial capacity of the JWKS cache.
	CacheMetadataJwksInitialCapacity int32 `json:"cacheMetadataJwksInitialCapacity,omitempty"`

	// Specifies the maximum capacity of the JWKS cache.
	CacheMetadataJwksMaximumSize int32 `json:"cacheMetadataJwksMaximumSize,omitempty"`

	// Specifies the duration after a JWK is not accessed from the cache that it expires.
	// By default this is disabled.
	CacheMetadataJwksExpireAfterAccess time.Duration `json:"cacheMetadataJwksExpireAfterAccess,omitempty"`

	// Specifies the duration after a JWK is cached, that is, it expires and has to be loaded again.
	// +kubebuilder:default=5m
	CacheMetadataJwksExpireAfterWrite time.Duration `json:"cacheMetadataJwksExpireAfterWrite,omitempty"`

	// Specifies to disable the Database API administration related services.
	// Only applicable when Database API is enabled.
	DatabaseApiManagementServicesDisabled bool `json:"databaseApiManagementServicesDisabled,omitempty"`

	// Specifies how long to wait before retrying an invalid pool.
	// Default: 15m
	DbInvalidPoolTimeout time.Duration `json:"dbInvalidPoolTimeout,omitempty"`

	// Specifies the maximum join nesting depth limit for GraphQL queries.
	// Defaults to 5.
	FeatureGrahpqlMaxNestingDepth int32 `json:"featureGrahpqlMaxNestingDepth,omitempty"`

	// Specifies the name of the HTTP request header that uniquely identifies the request end to end as
	// it passes through the various layers of the application stack.
	// In Oracle this header is commonly referred to as the ECID (Entity Context ID).
	RequestTraceHeaderName string `json:"requestTraceHeaderName,omitempty"`

	// Specifies the maximum number of unsuccessful password attempts allowed.
	// Enabled by setting a positive integer value.
	// Defaults to -1.
	SecurityCredentialsAttempts int32 `json:"securityCredentialsAttempts,omitempty"`

	// Specifies the file where credentials are stored.
	SecurityCredentialsFile string `json:"securityCredentialsFile,omitempty"`

	// Specifies the period to lock the account that has exceeded maximum attempts.
	// Defaults to 10m (10 minutes)
	SecurityCredentialsLockTime time.Duration `json:"securityCredentialsLockTime,omitempty"`

	// Specifies the path to the folder to store HTTP request access logs.
	// If not specified, then no access log is generated.
	StandaloneAccessLog string `json:"standaloneAccessLog,omitempty"`

	// Specifies the comma separated list of host names or IP addresses to identify a specific network
	// interface on which to listen.
	// Default 0.0.0.0
	StandaloneBinds net.IP `json:"standaloneBinds,omitempty"`

	// Specifies the context path where ords is located. Defaults to /ords
	StandaloneContextPath string `json:"standaloneContextPath,omitempty"`

	// Points to the location where static resources to be served under the / root server path are located.
	StandaloneDocRoot string `json:"standaloneDocRoot,omitempty"`

	// Specifies the HTTP listen port.
	// Default: 8080
	StandaloneHttpPort int32 `json:"standaloneHttpPort,omitempty" protobuf:"varint,3,opt,name=standalonehttpport"`

	// Specifies the SSL certificate path.
	// If you are providing the SSL certificate, then you must specify the certificate location.
	StandaloneHttpsCert string `json:"standaloneHttpsCert,omitempty"`

	// Specifies the SSL certificate key path.
	// If you are providing the SSL certificate, you must specify the certificate key location.
	StandaloneHttpsCertKey string `json:"standaloneHttpsCertKey,omitempty"`

	// Specifies the SSL certificate hostname.
	StandaloneHttpsHost string `json:"standaloneHttpsHost,omitempty"`

	// Specifies the HTTPS listen port.
	// Default: 8443
	StandaloneHttpsPort int32 `json:"standaloneHttpsPort,omitempty"`

	// Specifies the Context path where APEX static resources are located.
	// Default: /i
	StandaloneStaticContextPath string `json:"standaloneStaticContextPath,omitempty"`

	// Specifies the path to the folder containing static resources required by APEX.
	StandaloneStaticPath string `json:"standaloneStaticPath,omitempty"`

	// Specifies the period for Standalone Mode to wait until it is gracefully shutdown.
	// Default: 10s (10 seconds)
	StandaloneStopTimeout int32 `json:"standaloneStopTimeout,omitempty"`

	// Specifies the setting to determine for how long a metadata record remains in the cache.
	// Longer duration means, it takes longer to view the applied changes.
	// The formats accepted are based on the ISO-8601 duration format.
	CacheMetadataTimeout string `json:"cacheMetadataTimeout,omitempty"`

	// Specifies the setting to enable or disable metadata caching.
	CacheMetadataEnabled bool `json:"cacheMetadataEnabled,omitempty"`

	// Specifies whether the Database API is enabled.
	DatabaseApiEnabled bool `json:"databaseApiEnabled,omitempty"`

	// Specifies whether to display error messages on the browser.
	DebugPrintDebugToScreen bool `json:"debugPrintDebugToScreen,omitempty"`

	// Specifies how the HTTP error responses must be formatted.
	// html - Force all responses to be in HTML format
	// json - Force all responses to be in JSON format
	// auto - Automatically determines most appropriate format for the request (default).
	ErrorResponseFormat string `json:"errorResponseFormat,omitempty"`

	// Specifies the path to a folder that contains the custom error page.
	ErrorExternalPath string `json:"errorExternalPath,omitempty"`

	// Specifies the Internet Content Adaptation Protocol (ICAP) port to virus scan files.
	// Either icap.port or icap.secure.port are required to have a value.
	IcapPort int32 `json:"icapPort,omitempty"`

	// Specifies the Internet Content Adaptation Protocol (ICAP) port to virus scan files.
	// Either icap.port or icap.secure.port are required to have a value.
	// If values for both icap.port and icap.secure.port are provided, then the value of icap.port is ignored.
	IcapSecurePort int32 `json:"icapSecurePort,omitempty"`

	// Specifies the Internet Content Adaptation Protocol (ICAP) server name or IP address to virus scan files.
	// The icap.server is required to have a value.
	IcapServer string `json:"icapServer,omitempty"`

	// Specifies whether procedures are to be logged.
	LogProcedure bool `json:"logProcedure,omitempty"`

	// If this value is set to true, then the Oracle REST Data Services internal exclusion list is not enforced.
	// Oracle recommends that you do not set this value to true.
	SecurityDisableDefaultExclusionList bool `json:"securityDisableDefaultExclusionList,omitempty"`

	// Specifies a pattern for procedures, packages, or schema names which are forbidden to be directly executed from a browser.
	SecurityExclusionList string `json:"securityExclusionList,omitempty"`

	// Specifies a pattern for procedures, packages, or schema names which are allowed to be directly executed from a browser.
	SecurityInclusionList string `json:"securityInclusionList,omitempty"`

	// Specifies the maximum number of cached procedure validations.
	// Set this value to 0 to force the validation procedure to be invoked on each request.
	// Defaults to 2000.
	SecurityMaxEntries int32 `json:"securityMaxEntries,omitempty"`

	// Specifies whether HTTPS is available in your environment.
	SecurityVerifySSL bool `json:"securityVerifySSL,omitempty"`
}

// RestDataServicesStatus defines the observed state of RestDataServices
type RestDataServicesStatus struct {
	Image    string `json:"image,omitempty"`
	Replicas int32  `json:"replicas,omitempty"`

	// Conditions store the status conditions of the ORDS instances
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
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
