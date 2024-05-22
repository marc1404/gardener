//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

// Code generated by conversion-gen. DO NOT EDIT.

package v1

import (
	unsafe "unsafe"

	core "github.com/gardener/gardener/pkg/apis/core"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	conversion "k8s.io/apimachinery/pkg/conversion"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

func init() {
	localSchemeBuilder.Register(RegisterConversions)
}

// RegisterConversions adds conversion functions to the given scheme.
// Public to allow building arbitrary schemes.
func RegisterConversions(s *runtime.Scheme) error {
	if err := s.AddGeneratedConversionFunc((*ControllerDeploymentList)(nil), (*core.ControllerDeploymentList)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1_ControllerDeploymentList_To_core_ControllerDeploymentList(a.(*ControllerDeploymentList), b.(*core.ControllerDeploymentList), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*core.ControllerDeploymentList)(nil), (*ControllerDeploymentList)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_core_ControllerDeploymentList_To_v1_ControllerDeploymentList(a.(*core.ControllerDeploymentList), b.(*ControllerDeploymentList), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*HelmControllerDeployment)(nil), (*core.HelmControllerDeployment)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1_HelmControllerDeployment_To_core_HelmControllerDeployment(a.(*HelmControllerDeployment), b.(*core.HelmControllerDeployment), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*core.HelmControllerDeployment)(nil), (*HelmControllerDeployment)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_core_HelmControllerDeployment_To_v1_HelmControllerDeployment(a.(*core.HelmControllerDeployment), b.(*HelmControllerDeployment), scope)
	}); err != nil {
		return err
	}
	if err := s.AddConversionFunc((*core.ControllerDeployment)(nil), (*ControllerDeployment)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_core_ControllerDeployment_To_v1_ControllerDeployment(a.(*core.ControllerDeployment), b.(*ControllerDeployment), scope)
	}); err != nil {
		return err
	}
	if err := s.AddConversionFunc((*ControllerDeployment)(nil), (*core.ControllerDeployment)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1_ControllerDeployment_To_core_ControllerDeployment(a.(*ControllerDeployment), b.(*core.ControllerDeployment), scope)
	}); err != nil {
		return err
	}
	return nil
}

func autoConvert_v1_ControllerDeployment_To_core_ControllerDeployment(in *ControllerDeployment, out *core.ControllerDeployment, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Helm = (*core.HelmControllerDeployment)(unsafe.Pointer(in.Helm))
	return nil
}

func autoConvert_core_ControllerDeployment_To_v1_ControllerDeployment(in *core.ControllerDeployment, out *ControllerDeployment, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	// WARNING: in.Type requires manual conversion: does not exist in peer-type
	// WARNING: in.ProviderConfig requires manual conversion: does not exist in peer-type
	out.Helm = (*HelmControllerDeployment)(unsafe.Pointer(in.Helm))
	return nil
}

func autoConvert_v1_ControllerDeploymentList_To_core_ControllerDeploymentList(in *ControllerDeploymentList, out *core.ControllerDeploymentList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]core.ControllerDeployment, len(*in))
		for i := range *in {
			if err := Convert_v1_ControllerDeployment_To_core_ControllerDeployment(&(*in)[i], &(*out)[i], s); err != nil {
				return err
			}
		}
	} else {
		out.Items = nil
	}
	return nil
}

// Convert_v1_ControllerDeploymentList_To_core_ControllerDeploymentList is an autogenerated conversion function.
func Convert_v1_ControllerDeploymentList_To_core_ControllerDeploymentList(in *ControllerDeploymentList, out *core.ControllerDeploymentList, s conversion.Scope) error {
	return autoConvert_v1_ControllerDeploymentList_To_core_ControllerDeploymentList(in, out, s)
}

func autoConvert_core_ControllerDeploymentList_To_v1_ControllerDeploymentList(in *core.ControllerDeploymentList, out *ControllerDeploymentList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ControllerDeployment, len(*in))
		for i := range *in {
			if err := Convert_core_ControllerDeployment_To_v1_ControllerDeployment(&(*in)[i], &(*out)[i], s); err != nil {
				return err
			}
		}
	} else {
		out.Items = nil
	}
	return nil
}

// Convert_core_ControllerDeploymentList_To_v1_ControllerDeploymentList is an autogenerated conversion function.
func Convert_core_ControllerDeploymentList_To_v1_ControllerDeploymentList(in *core.ControllerDeploymentList, out *ControllerDeploymentList, s conversion.Scope) error {
	return autoConvert_core_ControllerDeploymentList_To_v1_ControllerDeploymentList(in, out, s)
}

func autoConvert_v1_HelmControllerDeployment_To_core_HelmControllerDeployment(in *HelmControllerDeployment, out *core.HelmControllerDeployment, s conversion.Scope) error {
	out.RawChart = *(*[]byte)(unsafe.Pointer(&in.RawChart))
	out.Values = (*apiextensionsv1.JSON)(unsafe.Pointer(in.Values))
	return nil
}

// Convert_v1_HelmControllerDeployment_To_core_HelmControllerDeployment is an autogenerated conversion function.
func Convert_v1_HelmControllerDeployment_To_core_HelmControllerDeployment(in *HelmControllerDeployment, out *core.HelmControllerDeployment, s conversion.Scope) error {
	return autoConvert_v1_HelmControllerDeployment_To_core_HelmControllerDeployment(in, out, s)
}

func autoConvert_core_HelmControllerDeployment_To_v1_HelmControllerDeployment(in *core.HelmControllerDeployment, out *HelmControllerDeployment, s conversion.Scope) error {
	out.RawChart = *(*[]byte)(unsafe.Pointer(&in.RawChart))
	out.Values = (*apiextensionsv1.JSON)(unsafe.Pointer(in.Values))
	return nil
}

// Convert_core_HelmControllerDeployment_To_v1_HelmControllerDeployment is an autogenerated conversion function.
func Convert_core_HelmControllerDeployment_To_v1_HelmControllerDeployment(in *core.HelmControllerDeployment, out *HelmControllerDeployment, s conversion.Scope) error {
	return autoConvert_core_HelmControllerDeployment_To_v1_HelmControllerDeployment(in, out, s)
}
