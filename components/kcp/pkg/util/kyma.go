package util

import (
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func NewKymaUnstructured() *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "operator.kyma-project.io",
		Version: "v1beta2",
		Kind:    "Kyma",
	})
	return u
}

func GetKymaModuleState(k *unstructured.Unstructured, moduleName string) KymaModuleState {
	modules, exists, err := unstructured.NestedSlice(k.Object, "status", "modules")
	if !exists || err != nil {
		return KymaModuleStateNotPresent
	}
	for _, m := range modules {
		mm, ok := m.(map[string]interface{})
		if !ok {
			return KymaModuleStateNotPresent
		}
		name, exists, err := unstructured.NestedString(mm, "name")
		if !exists || err != nil {
			return KymaModuleStateNotPresent
		}
		if name == moduleName {
			val, exists, err := unstructured.NestedString(mm, "state")
			if !exists || err != nil {
				return KymaModuleStateNotPresent
			}
			return KymaModuleState(val)
		}
	}

	return KymaModuleStateNotPresent
}

func SetKymaModuleState(k *unstructured.Unstructured, moduleName string, state KymaModuleState) error {
	modules, exists, err := unstructured.NestedSlice(k.Object, "status", "modules")
	if err != nil {
		return err
	}
	if !exists {
		modules = []interface{}{}
		err = unstructured.SetNestedSlice(k.Object, modules, "status", "modules")
		if err != nil {
			return nil
		}
	}

	for idx, m := range modules {
		mm, ok := m.(map[string]interface{})
		if !ok {
			return fmt.Errorf("kyma CR module #%d is not a map", idx)
		}

		name, exists, err := unstructured.NestedString(mm, "name")
		if err != nil {
			return err
		}
		if exists && name == moduleName {
			return unstructured.SetNestedField(mm, string(state), "state")
		}
	}

	modules = append(modules, map[string]interface{}{
		"name":  moduleName,
		"state": string(state),
	})

	return unstructured.SetNestedSlice(k.Object, modules, "status", "modules")
}

// https://github.com/kyma-project/lifecycle-manager/blob/main/api/shared/state.go

// KymaModuleState the state of the modul in the Kyma CR
type KymaModuleState string

// Valid States.
const (
	KymaModuleStateNotPresent KymaModuleState = ""

	// KymaModuleStateReady signifies CustomObject is ready and has been installed successfully.
	KymaModuleStateReady KymaModuleState = "Ready"

	// KymaModuleStateProcessing signifies CustomObject is reconciling and is in the process of installation.
	// Processing can also signal that the Installation previously encountered an error and is now recovering.
	KymaModuleStateProcessing KymaModuleState = "Processing"

	// KymaModuleStateError signifies an error for CustomObject. This signifies that the Installation
	// process encountered an error.
	// Contrary to Processing, it can be expected that this state should change on the next retry.
	KymaModuleStateError KymaModuleState = "Error"

	// KymaModuleStateDeleting signifies CustomObject is being deleted. This is the state that is used
	// when a deletionTimestamp was detected and Finalizers are picked up.
	KymaModuleStateDeleting KymaModuleState = "Deleting"

	// KymaModuleStateWarning signifies specified resource has been deployed, but cannot be used due to misconfiguration,
	// usually it means that user interaction is required.
	KymaModuleStateWarning KymaModuleState = "Warning"
)
