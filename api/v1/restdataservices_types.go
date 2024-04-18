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

package v1

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RestDataServicesSpec defines the desired state of RestDataServices
// +kubebuilder:resource:shortName="ords"
type RestDataServicesSpec struct {
	// +kubebuilder:validation:Enum=Deployment;StatefulSet;DaemonSet
	WorkloadType string `json:"workloadType,omitempty"`
	// +kubebuilder:validation:Minimum=1
	Replicas int32 `json:"replicas,omitempty"`
	// Specifies whether to restart pods when Global or Pool configurations change
	AutoRestart      bool              `json:"autoRestart,omitempty"`
	Image            string            `json:"image" protobuf:"bytes,2,opt,name=image"`
	ImagePullPolicy  corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	ImagePullSecrets string            `json:"imagePullSecrets,omitempty"`
	// Contains settings that are configured across the entire ORDS instance.
	GlobalSettings GlobalSettings `json:"globalSettings"`
	// Contains settings for individual pools/databases
	PoolSettings []*PoolSettings `json:"poolSettings,omitempty"`
	// +k8s:openapi-gen=true
}

// RestDataServicesStatus defines the observed state of RestDataServices
type RestDataServicesStatus struct {
	Image    string `json:"image,omitempty"`
	Replicas int32  `json:"replicas,omitempty"`

	// Conditions store the status conditions of the ORDS instances
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

type GlobalSettings struct {
	// Specifies the duration after a GraphQL schema is not accessed from the cache that it expires.
	CacheMetadataGraphqlExpireAfterAccess *time.Duration `json:"cacheMetadataGraphqlExpireAfterAccess,omitempty"`

	// Specifies the setting to enable or disable JWKS caching.
	CacheMetadataJwksEnabled *bool `json:"cacheMetadataJwksEnabled,omitempty"`

	// Specifies the initial capacity of the JWKS cache.
	CacheMetadataJwksInitialCapacity *int32 `json:"cacheMetadataJwksInitialCapacity,omitempty"`

	// Specifies the maximum capacity of the JWKS cache.
	CacheMetadataJwksMaximumSize *int32 `json:"cacheMetadataJwksMaximumSize,omitempty"`

	// Specifies the duration after a JWK is not accessed from the cache that it expires.
	// By default this is disabled.
	CacheMetadataJwksExpireAfterAccess *time.Duration `json:"cacheMetadataJwksExpireAfterAccess,omitempty"`

	// Specifies the duration after a JWK is cached, that is, it expires and has to be loaded again.
	CacheMetadataJwksExpireAfterWrite *time.Duration `json:"cacheMetadataJwksExpireAfterWrite,omitempty"`

	// Specifies to disable the Database API administration related services.
	// Only applicable when Database API is enabled.
	DatabaseApiManagementServicesDisabled *bool `json:"databaseApiManagementServicesDisabled,omitempty"`

	// Specifies how long to wait before retrying an invalid pool.
	// Default: 15m
	DbInvalidPoolTimeout *time.Duration `json:"dbInvalidPoolTimeout,omitempty"`

	// Specifies the maximum join nesting depth limit for GraphQL queries.
	// +kubebuilder:default:=5
	FeatureGrahpqlMaxNestingDepth *int32 `json:"featureGrahpqlMaxNestingDepth,omitempty"`

	// Specifies the name of the HTTP request header that uniquely identifies the request end to end as
	// it passes through the various layers of the application stack.
	// In Oracle this header is commonly referred to as the ECID (Entity Context ID).
	RequestTraceHeaderName string `json:"requestTraceHeaderName,omitempty"`

	// Specifies the maximum number of unsuccessful password attempts allowed.
	// Enabled by setting a positive integer value.
	// +kubebuilder:default:=-1
	SecurityCredentialsAttempts *int32 `json:"securityCredentialsAttempts,omitempty"`

	// Specifies the period to lock the account that has exceeded maximum attempts.
	// Defaults to 10m (10 minutes)
	SecurityCredentialsLockTime *time.Duration `json:"securityCredentialsLockTime,omitempty"`

	// Specifies the comma separated list of host names or IP addresses to identify a specific network
	// interface on which to listen.
	// +kubebuilder:default:="0.0.0.0"
	StandaloneBinds string `json:"standaloneBinds,omitempty"`

	// Specifies the HTTP listen port.
	/// +kubebuilder:default:=8080
	StandaloneHttpPort *int32 `json:"standaloneHttpPort,omitempty" protobuf:"varint,3,opt,name=standalonehttpport"`

	// Specifies the SSL certificate hostname.
	StandaloneHttpsHost string `json:"standaloneHttpsHost,omitempty"`

	// Specifies the HTTPS listen port.
	// +kubebuilder:default:=8443
	StandaloneHttpsPort *int32 `json:"standaloneHttpsPort,omitempty"`

	// Specifies the period for Standalone Mode to wait until it is gracefully shutdown.
	// Default: 10s (10 seconds)
	StandaloneStopTimeout *int32 `json:"standaloneStopTimeout,omitempty"`

	// Specifies the setting to determine for how long a metadata record remains in the cache.
	// Longer duration means, it takes longer to view the applied changes.
	// The formats accepted are based on the ISO-8601 duration format.
	CacheMetadataTimeout string `json:"cacheMetadataTimeout,omitempty"`

	// Specifies the setting to enable or disable metadata caching.
	CacheMetadataEnabled *bool `json:"cacheMetadataEnabled,omitempty"`

	// Specifies whether the Database API is enabled.
	DatabaseApiEnabled *bool `json:"databaseApiEnabled,omitempty"`

	// Specifies whether to display error messages on the browser.
	DebugPrintDebugToScreen *bool `json:"debugPrintDebugToScreen,omitempty"`

	// Specifies how the HTTP error responses must be formatted.
	// html - Force all responses to be in HTML format
	// json - Force all responses to be in JSON format
	// auto - Automatically determines most appropriate format for the request (default).
	// +kubebuilder:default:="auto"
	ErrorResponseFormat string `json:"errorResponseFormat,omitempty"`

	// Specifies the Internet Content Adaptation Protocol (ICAP) port to virus scan files.
	// Either icap.port or icap.secure.port are required to have a value.
	IcapPort *int32 `json:"icapPort,omitempty"`

	// Specifies the Internet Content Adaptation Protocol (ICAP) port to virus scan files.
	// Either icap.port or icap.secure.port are required to have a value.
	// If values for both icap.port and icap.secure.port are provided, then the value of icap.port is ignored.
	IcapSecurePort *int32 `json:"icapSecurePort,omitempty"`

	// Specifies the Internet Content Adaptation Protocol (ICAP) server name or IP address to virus scan files.
	// The icap.server is required to have a value.
	IcapServer string `json:"icapServer,omitempty"`

	// Specifies whether procedures are to be logged.
	LogProcedure *bool `json:"logProcedure,omitempty"`

	// If this value is set to true, then the Oracle REST Data Services internal exclusion list is not enforced.
	// Oracle recommends that you do not set this value to true.
	SecurityDisableDefaultExclusionList *bool `json:"securityDisableDefaultExclusionList,omitempty"`

	// Specifies a pattern for procedures, packages, or schema names which are forbidden to be directly executed from a browser.
	SecurityExclusionList string `json:"securityExclusionList,omitempty"`

	// Specifies a pattern for procedures, packages, or schema names which are allowed to be directly executed from a browser.
	SecurityInclusionList string `json:"securityInclusionList,omitempty"`

	// Specifies the maximum number of cached procedure validations.
	// Set this value to 0 to force the validation procedure to be invoked on each request.
	// +kubebuilder:default:=2000.
	SecurityMaxEntries *int32 `json:"securityMaxEntries,omitempty"`

	// Specifies whether HTTPS is available in your environment.
	SecurityVerifySSL *bool `json:"securityVerifySSL,omitempty"`

	// Specifies the context path where ords is located.
	// +kubebuilder:default:="/ords"
	StandaloneContextPath string `json:"-"`

	// Points to the location where static resources to be served under the / root server path are located.
	StandaloneDocRoot string `json:"-"`

	// Specifies the file where credentials are stored.
	SecurityCredentialsFile string `json:"-"`

	// Specifies the path to a folder that contains the custom error page.
	ErrorExternalPath string `json:"-"`

	// Specifies the Context path where APEX static resources are located.
	// +kubebuilder:default:="/i"
	StandaloneStaticContextPath string `json:"-"`

	// Specifies the path to the folder containing static resources required by APEX.
	StandaloneStaticPath string `json:"-"`

	// Specifies the SSL certificate path.
	// If you are providing the SSL certificate, then you must specify the certificate location.
	StandaloneHttpsCert string `json:"-"`

	// Specifies the SSL certificate key path.
	// If you are providing the SSL certificate, you must specify the certificate key location.
	StandaloneHttpsCertKey string `json:"-"`

	// Specifies the path to the folder to store HTTP request access logs.
	// If not specified, then no access log is generated.
	StandaloneAccessLog string `json:"-"`
}

type PoolSettings struct {
	// Specifies the Pool Name
	PoolName string `json:"poolName"`

	// Specifies the Secret with the dbUser (ORDS_PUBLIC_USER) and dbPassword values
	// for the connection.
	DbAuthSecret UsernamePasswordSecret `json:"dbAuthSecret"`

	// Specifies the Secret with the dbAdminUser (SYS AS SYSDBA) and dbAdminPassword values
	// for the database account that ORDS uses for administration operations in the database.
	DbAdminAuthSecret UsernamePasswordSecret `json:"dbAdminAuthSecret,omitempty"`

	// Specifies the Secret with the dbCdbAdminUser (SYS AS SYSDBA) and dbCdbAdminPassword values
	// Specifies the username for the database account that ORDS uses for the Pluggable Database Lifecycle Management.
	DbCdbAdminAuthSecret UsernamePasswordSecret `json:"dbCdbAdminAuthSecret,omitempty"`

	// Specifies the comma delimited list of additional roles to assign authenticated APEX administrator type users.
	ApexSecurityAdministratorRoles string `json:"apexSecurityAdministratorRoles,omitempty"`

	// Specifies the comma delimited list of additional roles to assign authenticated regular APEX users.
	ApexSecurityUserRoles string `json:"ApexSecurityUserRoles,omitempty"`

	// specifies a configuration setting for AutoUpgrade.jar location.
	AutoupgradeApiAulocation string `json:"AutoupgradeApiAulocation,omitempty"`

	// Specifies a configuration setting to enable AutoUpgrade REST API features.
	AutoupgradeApiEnabled *bool `json:"autoupgradeApiEnabled,omitempty"`

	// Specifies a configuration setting for AutoUpgrade REST API JVM location.
	AutoupgradeApiJvmlocation string `json:"AutoupgradeApiJvmlocation,omitempty"`

	// Specifies a configuration setting for AutoUpgrade REST API log location.
	AutoupgradeApiLoglocation string `json:"AutoupgradeApiLoglocation,omitempty"`

	// Specifies the username for the database account that ORDS uses for administration operations in the database.
	// Replaced by: DbAdminAuthSecret UsernamePasswordSecret
	// DbAdminUser string `json:"DbAdminUser,omitempty"`

	// Specifies the password for the database account that ORDS uses for administration operations in the database.
	// Replaced by: DbAdminAuthSecret UsernamePasswordSecret
	// DbAdminUserPassword struct{} `json:"DbAdminUserPassword,omitempty"`

	// Specifies the username for the database account that ORDS uses for the Pluggable Database Lifecycle Management.
	// Replaced by: DbCdbAdminAuthSecret UsernamePasswordSecret
	// DbCdbAdminUser string `json:"dbCdbAdminUser,omitempty"`

	// Specifies the password for the database account that ORDS uses for the Pluggable Database Lifecycle Management.
	// Replaced by: DbCdbAdminAuthSecret UsernamePasswordSecret
	// DbCdbAdminUserPassword struct{} `json:"dbCdbAdminUserPassword,omitempty"`

	// Specifies the source for database credentials when creating a direct connection for running SQL statements.
	// Value can be one of pool or request.
	// If the value is pool, then the credentials defined in this pool is used to create a JDBC connection.
	// If the value request is used, then the credentials in the request is used to create a JDBC connection and if successful, grants the requestor SQL Developer role.
	// +kubebuilder:default:="pool"
	DbCredentialsSource string `json:"DbCredentialsSource,omitempty"`

	// Indicates how long to wait to gracefully destroy a pool before moving to forcefully destroy all connections including borrowed ones.
	// Default: 5m
	DbPoolDestroyTimeout *time.Duration `json:"dbPoolDestroyTimeout,omitempty"`

	// Specifies the wallet archive (provided in BASE64 encoding) containing connection details for the pool.
	DbWalletZip string `json:"dbWalletZip,omitempty"`

	// Specifies the path to a wallet archive containing connection details for the pool.
	DbWalletZipPath string `json:"DbWalletZipPath,omitempty"`

	// Specifies the service name in the wallet archive for the pool.
	DbWalletZipService string `json:"dbWalletZipService,omitempty"`

	// Specifies to enable tracking of JDBC resources.
	// If not released causes in resource leaks or exhaustion in the database.
	// Tracking imposes a performance overhead.
	DebugTrackResources *bool `json:"debugTrackResources,omitempty"`

	// Specifies to disable the Open Service Broker services available for the pool.
	FeatureOpenservicebrokerExclude *bool `json:"featureOpenservicebrokerExclude,omitempty"`

	// Specifies to enable the Database Actions feature.
	FeatureSdw *bool `json:"featureSdw,omitempty"`

	// Specifies a comma separated list of HTTP Cookies to exclude when initializing an Oracle Web Agent environment.
	HttpCookieFilter string `json:"httpCookieFilter,omitempty"`

	// Identifies the database role that indicates that the database user must get the SQL Administrator role.
	JdbcAuthAdminRole string `json:"JdbcAuthAdminRole,omitempty"`

	// Specifies how a pooled JDBC connection and corresponding database session, is released when a request has been processed.
	// +kubebuilder:default:="RECYCLE"
	JdbCleanupMode string `json:"jdbCleanupMode,omitempty"`

	// If it is true, then it causes a trace of the SQL statements performed by Oracle Web Agent to be echoed to the log.
	OwaTraceSql *bool `json:"owaTraceSql,omitempty"`

	// Indicates if the PL/SQL Gateway functionality should be available for a pool or not.
	// Value can be one of disabled, direct, or proxied.
	// If the value is direct, then the pool serves the PL/SQL Gateway requests directly.
	// If the value is PLSQL_GATEWAY_CONFIG, view is used to determine the user to whom to proxy.
	PlsqlGatewayMode string `json:"PlsqlGatewayMode,omitempty"`

	// Specifies whether the JWT Profile authentication is available. Supported values:
	// +kubebuilder:default:=true
	SecurityJwtProfileEnabled *bool `json:"securityJwtProfileEnabled,omitempty"`

	// Specifies the maximum number of bytes read from the JWK url.
	// Default 100000 bytes.
	SecurityJwksSize *int32 `json:"securityJwksSize,omitempty"`

	// Specifies the maximum amount of time before timing-out when accessing a JWK url.
	// Default is 5 seconds.
	SecurityJwksConnectionTimeout *time.Duration `json:"securityJwksConnectionTimeout,omitempty"`

	// Specifies the maximum amount of time reading a response from the JWK url before timing-out.
	// Default is 5 seconds.
	SecurityJwksReadTimeout *time.Duration `json:"securityJwksReadTimeout,omitempty"`

	// Specifies the minimum interval between refreshing the JWK cached value.
	SecurityJwksRefreshInterval *time.Duration `json:"securityJwksRefreshInterval,omitempty"`

	// Specifies the maximum skew the JWT time claims are accepted.
	// This is useful if the clock on the JWT issuer and ORDS differs by a few seconds.
	// By default, it is disabled.
	SecurityJwtAllowedSkew *time.Duration `json:"securityJwtAllowedSkew,omitempty"`

	// Specifies the maximum allowed age of a JWT in seconds, regardless of expired claim.
	// The age of the JWT is taken from the JWT issued at claim.
	// By default, it is disabled.
	SecurityJwtAllowedAge *time.Duration `json:"securityJwtAllowedAge,omitempty"`

	// Indicates the type of security.requestValidationFunction: javascript or plsql.
	/// +kubebuilder:default:="plsql"
	SecurityValidationFunctionType string `json:"securityValidationFunctionType,omitempty"`

	// The type of connection. Supported values: basic, tns, customurl
	DbConnectionType string `json:"dbConnectionType,omitempty"`

	// Specifies the JDBC URL connection to connect to the database.
	DbCustomURL string `json:"dbCustomURL,omitempty"`

	// Specifies the host system for the Oracle database.
	DbHostname string `json:"dbHostname,omitempty"`

	// Specifies the password of the specified database user.
	// Include an exclamation at the beginning of the password so that it can be stored encrypted.
	// Replaced by: DbAuthSecret UsernamePasswordSecret `json:"dbAuthSecret"`
	// DbPassword struct{} `json:"dbPassword,omitempty"`

	// Specifies the database listener port.
	DbPort *int32 `json:"DbPort,omitempty"`

	// Specifies the network service name of the database.
	DbServicename string `json:"dbServicename,omitempty"`

	// Specifies that the pool points to a CDB, and that the PDBs connected to that CDB should be made addressable
	// by Oracle REST Data Services
	DbServiceNameSuffix string `json:"dbServiceNameSuffix,omitempty"`

	// Specifies the name of the database.
	DbSid string `json:"dbSid,omitempty"`

	// Specifies the TNS alias name that matches the name in the tnsnames.ora file.
	DbTnsAliasName string `json:"dbTnsAliasName,omitempty"`

	// The directory location of your tnsnames.ora file.
	DbTnsDirectory string `json:"dbTnsDirectory,omitempty"`

	// Specifies the name of the database user for the connection.
	// Replaced by: DbAuthSecret UsernamePasswordSecret `json:"dbAuthSecret"`
	// DbUsername string `json:"dbUsername,omitempty"`

	// Specifies the JDBC driver type. Supported values: thin, oci8
	/// +kubebuilder:default:="thin"
	JdbcDriverType string `json:"jdbcDriverType,omitempty"`

	// Specifies how long an available connection can remain idle before it is closed. The inactivity connection timeout is in seconds.
	// +kubebuilder:default:=1800.
	JdbcInactivityTimeout *int32 `json:"jdbcInactivityTimeout,omitempty"`

	// Specifies the initial size for the number of connections that will be created.
	// The default is low, and should probably be set higher in most production environments.
	// +kubebuilder:default:=10
	JdbcInitialLimit *int32 `json:"jdbcInitialLimit,omitempty"`

	// Specifies the maximum number of times to reuse a connection before it is discarded and replaced with a new connection.
	// +kubebuilder:default:=1000.
	JdbcMaxConnectionReuseCount *int32 `json:"jdbcMaxConnectionReuseCount,omitempty"`

	// Specifies the maximum number of connections.
	// Might be too low for some production environments.
	// +kubebuilder:default:=10
	JdbcMaxLimit *int32 `json:"jdbcMaxLimit,omitempty"`

	// Specifies if the PL/SQL Gateway calls can be authenticated using database users.
	// If the value is true then this feature is enabled. If the value is false, then this feature is disabled.
	// Oracle recommends not to use this feature.
	// This feature used only to facilitate customers migrating from mod_plsql.
	// +kubebuilder:default:=false
	JdbcAuthEnabled *bool `json:"jdbcAuthEnabled,omitempty"`

	// Specifies the maximum number of statements to cache for each connection.
	// +kubebuilder:default:=10
	JdbcMaxStatementsLimit *int32 `json:"jdbcMaxStatementsLimit,omitempty"`

	// Specifies the minimum number of connections.
	// +kubebuilder:default:=2
	JdbcMinLimit *int32 `json:"JdbcMinLimit,omitempty"`

	// Specifies a timeout period on a statement.
	// An abnormally long running query or script, executed by a request, may leave it in a hanging state unless a timeout is
	// set on the statement. Setting a timeout on the statement ensures that all the queries automatically timeout if
	// they are not completed within the specified time period.
	// +kubebuilder:default:=900
	JdbcStatementTimeout *int32 `json:"jdbcStatementTimeout,omitempty"`

	// Specifies the default page to display. The Oracle REST Data Services Landing Page.
	MiscDefaultPage string `json:"miscDefaultPage,omitempty"`

	// Specifies the maximum number of rows that will be returned from a query when processing a RESTful service
	// and that will be returned from a nested cursor in a result set.
	// Affects all RESTful services generated through a SQL query, regardless of whether the resource is paginated.
	// +kubebuilder:default:=10000
	MiscPaginationMaxRows *int32 `json:"miscPaginationMaxRows,omitempty"`

	// Specifies the procedure name(s) to execute after executing the procedure specified on the URL.
	// Multiple procedure names must be separated by commas.
	ProcedurePostProcess string `json:"procedurePostProcess,omitempty"`

	// Specifies the procedure name(s) to execute prior to executing the procedure specified on the URL.
	// Multiple procedure names must be separated by commas.
	ProcedurePreProcess string `json:"procedurePreProcess,omitempty"`

	// Specifies the function to be invoked prior to dispatching each Oracle REST Data Services based REST Service.
	// The function can perform configuration of the database session, perform additional validation or authorization of the request.
	// If the function returns true, then processing of the request continues.
	// If the function returns false, then processing of the request is aborted and an HTTP 403 Forbidden status is returned.
	ProcedureRestPreHook string `json:"procedureRestPreHook,omitempty"`

	// Specifies an authentication function to determine if the requested procedure in the URL should be allowed or disallowed for processing.
	// The function should return true if the procedure is allowed; otherwise, it should return false.
	// If it returns false, Oracle REST Data Services will return WWW-Authenticate in the response header.
	SecurityRequestAuthenticationFunction string `json:"securityRequestAuthenticationFunction,omitempty"`

	// Specifies a validation function to determine if the requested procedure in the URL should be allowed or disallowed for processing.
	// The function should return true if the procedure is allowed; otherwise, return false.
	SecurityRequestValidationFunction string `json:"securityRequestValidationFunction,omitempty"`

	// When using the SODA REST API, specifies the default number of documents returned for a GET request on a collection when a
	// limit is not specified in the URL. Must be a positive integer, or "unlimited" for no limit.
	// +kubebuilder:default:=100
	SodaDefaultLimit string `json:"sodaDefaultLimit,omitempty"`

	// When using the SODA REST API, specifies the maximum number of documents that will be returned for a GET request on a collection URL,
	// regardless of any limit specified in the URL. Must be a positive integer, or "unlimited" for no limit.
	// +kubebuilder:default:=1000
	SodaMaxLimit string `json:"SodaMaxLimit,omitempty"`

	// Specifies whether the REST-Enabled SQL service is active.
	// +kubebuilder:default:=false
	RestEnabledSqlActive *bool `json:"restEnabledSqlActive,omitempty"`
}

// Defines the secret containing Username/Password mapped to secretKey
// Replaces PoolSettings: DbUsername, DbPassword
type UsernamePasswordSecret struct {
	SecretName string `json:"secretName"`
	// +kubebuilder:default:="username"
	UsernameKey string `json:"usernameKey,omitempty"`
	// +kubebuilder:default:="password"
	PasswordKey string `json:"passwordKey,omitempty"`
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
