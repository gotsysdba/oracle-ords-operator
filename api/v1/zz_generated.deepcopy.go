//go:build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	timex "time"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GlobalSettings) DeepCopyInto(out *GlobalSettings) {
	*out = *in
	if in.CacheMetadataGraphqlExpireAfterAccess != nil {
		in, out := &in.CacheMetadataGraphqlExpireAfterAccess, &out.CacheMetadataGraphqlExpireAfterAccess
		*out = new(timex.Duration)
		**out = **in
	}
	if in.CacheMetadataJwksEnabled != nil {
		in, out := &in.CacheMetadataJwksEnabled, &out.CacheMetadataJwksEnabled
		*out = new(bool)
		**out = **in
	}
	if in.CacheMetadataJwksInitialCapacity != nil {
		in, out := &in.CacheMetadataJwksInitialCapacity, &out.CacheMetadataJwksInitialCapacity
		*out = new(int32)
		**out = **in
	}
	if in.CacheMetadataJwksMaximumSize != nil {
		in, out := &in.CacheMetadataJwksMaximumSize, &out.CacheMetadataJwksMaximumSize
		*out = new(int32)
		**out = **in
	}
	if in.CacheMetadataJwksExpireAfterAccess != nil {
		in, out := &in.CacheMetadataJwksExpireAfterAccess, &out.CacheMetadataJwksExpireAfterAccess
		*out = new(timex.Duration)
		**out = **in
	}
	if in.CacheMetadataJwksExpireAfterWrite != nil {
		in, out := &in.CacheMetadataJwksExpireAfterWrite, &out.CacheMetadataJwksExpireAfterWrite
		*out = new(timex.Duration)
		**out = **in
	}
	if in.DatabaseApiManagementServicesDisabled != nil {
		in, out := &in.DatabaseApiManagementServicesDisabled, &out.DatabaseApiManagementServicesDisabled
		*out = new(bool)
		**out = **in
	}
	if in.DbInvalidPoolTimeout != nil {
		in, out := &in.DbInvalidPoolTimeout, &out.DbInvalidPoolTimeout
		*out = new(timex.Duration)
		**out = **in
	}
	if in.FeatureGrahpqlMaxNestingDepth != nil {
		in, out := &in.FeatureGrahpqlMaxNestingDepth, &out.FeatureGrahpqlMaxNestingDepth
		*out = new(int32)
		**out = **in
	}
	if in.SecurityCredentialsAttempts != nil {
		in, out := &in.SecurityCredentialsAttempts, &out.SecurityCredentialsAttempts
		*out = new(int32)
		**out = **in
	}
	if in.SecurityCredentialsLockTime != nil {
		in, out := &in.SecurityCredentialsLockTime, &out.SecurityCredentialsLockTime
		*out = new(timex.Duration)
		**out = **in
	}
	if in.StandaloneHttpPort != nil {
		in, out := &in.StandaloneHttpPort, &out.StandaloneHttpPort
		*out = new(int32)
		**out = **in
	}
	if in.StandaloneHttpsPort != nil {
		in, out := &in.StandaloneHttpsPort, &out.StandaloneHttpsPort
		*out = new(int32)
		**out = **in
	}
	if in.StandaloneStopTimeout != nil {
		in, out := &in.StandaloneStopTimeout, &out.StandaloneStopTimeout
		*out = new(int32)
		**out = **in
	}
	if in.CacheMetadataEnabled != nil {
		in, out := &in.CacheMetadataEnabled, &out.CacheMetadataEnabled
		*out = new(bool)
		**out = **in
	}
	if in.DatabaseApiEnabled != nil {
		in, out := &in.DatabaseApiEnabled, &out.DatabaseApiEnabled
		*out = new(bool)
		**out = **in
	}
	if in.DebugPrintDebugToScreen != nil {
		in, out := &in.DebugPrintDebugToScreen, &out.DebugPrintDebugToScreen
		*out = new(bool)
		**out = **in
	}
	if in.IcapPort != nil {
		in, out := &in.IcapPort, &out.IcapPort
		*out = new(int32)
		**out = **in
	}
	if in.IcapSecurePort != nil {
		in, out := &in.IcapSecurePort, &out.IcapSecurePort
		*out = new(int32)
		**out = **in
	}
	if in.LogProcedure != nil {
		in, out := &in.LogProcedure, &out.LogProcedure
		*out = new(bool)
		**out = **in
	}
	if in.SecurityDisableDefaultExclusionList != nil {
		in, out := &in.SecurityDisableDefaultExclusionList, &out.SecurityDisableDefaultExclusionList
		*out = new(bool)
		**out = **in
	}
	if in.SecurityMaxEntries != nil {
		in, out := &in.SecurityMaxEntries, &out.SecurityMaxEntries
		*out = new(int32)
		**out = **in
	}
	if in.SecurityVerifySSL != nil {
		in, out := &in.SecurityVerifySSL, &out.SecurityVerifySSL
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GlobalSettings.
func (in *GlobalSettings) DeepCopy() *GlobalSettings {
	if in == nil {
		return nil
	}
	out := new(GlobalSettings)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PoolSettings) DeepCopyInto(out *PoolSettings) {
	*out = *in
	if in.AutoupgradeApiEnabled != nil {
		in, out := &in.AutoupgradeApiEnabled, &out.AutoupgradeApiEnabled
		*out = new(bool)
		**out = **in
	}
	if in.DbPoolDestroyTimeout != nil {
		in, out := &in.DbPoolDestroyTimeout, &out.DbPoolDestroyTimeout
		*out = new(timex.Duration)
		**out = **in
	}
	if in.DebugTrackResources != nil {
		in, out := &in.DebugTrackResources, &out.DebugTrackResources
		*out = new(bool)
		**out = **in
	}
	if in.FeatureOpenservicebrokerExclude != nil {
		in, out := &in.FeatureOpenservicebrokerExclude, &out.FeatureOpenservicebrokerExclude
		*out = new(bool)
		**out = **in
	}
	if in.FeatureSdw != nil {
		in, out := &in.FeatureSdw, &out.FeatureSdw
		*out = new(bool)
		**out = **in
	}
	if in.OwaTraceSql != nil {
		in, out := &in.OwaTraceSql, &out.OwaTraceSql
		*out = new(bool)
		**out = **in
	}
	if in.SecurityJwtProfileEnabled != nil {
		in, out := &in.SecurityJwtProfileEnabled, &out.SecurityJwtProfileEnabled
		*out = new(bool)
		**out = **in
	}
	if in.SecurityJwksSize != nil {
		in, out := &in.SecurityJwksSize, &out.SecurityJwksSize
		*out = new(int32)
		**out = **in
	}
	if in.SecurityJwksConnectionTimeout != nil {
		in, out := &in.SecurityJwksConnectionTimeout, &out.SecurityJwksConnectionTimeout
		*out = new(timex.Duration)
		**out = **in
	}
	if in.SecurityJwksReadTimeout != nil {
		in, out := &in.SecurityJwksReadTimeout, &out.SecurityJwksReadTimeout
		*out = new(timex.Duration)
		**out = **in
	}
	if in.SecurityJwksRefreshInterval != nil {
		in, out := &in.SecurityJwksRefreshInterval, &out.SecurityJwksRefreshInterval
		*out = new(timex.Duration)
		**out = **in
	}
	if in.SecurityJwtAllowedSkew != nil {
		in, out := &in.SecurityJwtAllowedSkew, &out.SecurityJwtAllowedSkew
		*out = new(timex.Duration)
		**out = **in
	}
	if in.SecurityJwtAllowedAge != nil {
		in, out := &in.SecurityJwtAllowedAge, &out.SecurityJwtAllowedAge
		*out = new(timex.Duration)
		**out = **in
	}
	if in.DbPort != nil {
		in, out := &in.DbPort, &out.DbPort
		*out = new(int32)
		**out = **in
	}
	if in.JdbcInactivityTimeout != nil {
		in, out := &in.JdbcInactivityTimeout, &out.JdbcInactivityTimeout
		*out = new(int32)
		**out = **in
	}
	if in.JdbcInitialLimit != nil {
		in, out := &in.JdbcInitialLimit, &out.JdbcInitialLimit
		*out = new(int32)
		**out = **in
	}
	if in.JdbcMaxConnectionReuseCount != nil {
		in, out := &in.JdbcMaxConnectionReuseCount, &out.JdbcMaxConnectionReuseCount
		*out = new(int32)
		**out = **in
	}
	if in.JdbcMaxLimit != nil {
		in, out := &in.JdbcMaxLimit, &out.JdbcMaxLimit
		*out = new(int32)
		**out = **in
	}
	if in.JdbcAuthEnabled != nil {
		in, out := &in.JdbcAuthEnabled, &out.JdbcAuthEnabled
		*out = new(bool)
		**out = **in
	}
	if in.JdbcMaxStatementsLimit != nil {
		in, out := &in.JdbcMaxStatementsLimit, &out.JdbcMaxStatementsLimit
		*out = new(int32)
		**out = **in
	}
	if in.JdbcMinLimit != nil {
		in, out := &in.JdbcMinLimit, &out.JdbcMinLimit
		*out = new(int32)
		**out = **in
	}
	if in.JdbcStatementTimeout != nil {
		in, out := &in.JdbcStatementTimeout, &out.JdbcStatementTimeout
		*out = new(int32)
		**out = **in
	}
	if in.MiscPaginationMaxRows != nil {
		in, out := &in.MiscPaginationMaxRows, &out.MiscPaginationMaxRows
		*out = new(int32)
		**out = **in
	}
	if in.RestEnabledSqlActive != nil {
		in, out := &in.RestEnabledSqlActive, &out.RestEnabledSqlActive
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PoolSettings.
func (in *PoolSettings) DeepCopy() *PoolSettings {
	if in == nil {
		return nil
	}
	out := new(PoolSettings)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RestDataServices) DeepCopyInto(out *RestDataServices) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RestDataServices.
func (in *RestDataServices) DeepCopy() *RestDataServices {
	if in == nil {
		return nil
	}
	out := new(RestDataServices)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RestDataServices) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RestDataServicesList) DeepCopyInto(out *RestDataServicesList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]RestDataServices, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RestDataServicesList.
func (in *RestDataServicesList) DeepCopy() *RestDataServicesList {
	if in == nil {
		return nil
	}
	out := new(RestDataServicesList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RestDataServicesList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RestDataServicesSpec) DeepCopyInto(out *RestDataServicesSpec) {
	*out = *in
	in.GlobalSettings.DeepCopyInto(&out.GlobalSettings)
	if in.PoolSettings != nil {
		in, out := &in.PoolSettings, &out.PoolSettings
		*out = make([]*PoolSettings, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(PoolSettings)
				(*in).DeepCopyInto(*out)
			}
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RestDataServicesSpec.
func (in *RestDataServicesSpec) DeepCopy() *RestDataServicesSpec {
	if in == nil {
		return nil
	}
	out := new(RestDataServicesSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RestDataServicesStatus) DeepCopyInto(out *RestDataServicesStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RestDataServicesStatus.
func (in *RestDataServicesStatus) DeepCopy() *RestDataServicesStatus {
	if in == nil {
		return nil
	}
	out := new(RestDataServicesStatus)
	in.DeepCopyInto(out)
	return out
}
