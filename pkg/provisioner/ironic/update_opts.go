package ironic

import (
	"fmt"
	"reflect"

	"github.com/go-logr/logr"

	"github.com/gophercloud/gophercloud/openstack/baremetal/v1/nodes"
)

type optionsData map[string]interface{}

func optionValueEqual(current, value interface{}) bool {
	switch curVal := current.(type) {
	case string:
		if newStr, ok := value.(string); ok {
			return curVal == newStr
		}
	case bool:
		if newBool, ok := value.(bool); ok {
			return curVal == newBool
		}
	case int:
		if newInt, ok := value.(int); ok {
			return curVal == newInt
		}
	case []interface{}:
		// newType could reasonably be either []interface{} or e.g. []string,
		// so we must use reflection.
		newType := reflect.TypeOf(value)
		switch newType.Kind() {
		case reflect.Slice, reflect.Array:
		default:
			return false
		}
		newList := reflect.ValueOf(value)
		if newList.Len() != len(curVal) {
			return false
		}
		for i, v := range curVal {
			if !optionValueEqual(newList.Index(i).Interface(), v) {
				return false
			}
		}
		return true
	case map[string]interface{}:
		// newType could reasonably be either map[string]interface{} or
		// e.g. map[string]string, so we must use reflection.
		newType := reflect.TypeOf(value)
		if newType.Kind() != reflect.Map ||
			newType.Key().Kind() != reflect.String {
			return false
		}
		newMap := reflect.ValueOf(value)
		if newMap.Len() != len(curVal) {
			return false
		}
		for k, v := range curVal {
			newV := newMap.MapIndex(reflect.ValueOf(k))
			if !(newV.IsValid() && optionValueEqual(newV.Interface(), v)) {
				return false
			}
		}
		return true
	}
	return false
}

func deref(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	if reflect.TypeOf(v).Kind() != reflect.Ptr {
		return v
	}
	if ptrVal := reflect.ValueOf(v); ptrVal.IsNil() {
		return nil
	} else {
		return ptrVal.Elem().Interface()
	}
}

func getUpdateOperation(name string, currentData map[string]interface{}, desiredValue interface{}, path string, log logr.Logger) *nodes.UpdateOperation {
	current, present := currentData[name]

	desiredValue = deref(desiredValue)
	if desiredValue != nil {
		if !(present && optionValueEqual(deref(current), desiredValue)) {
			if log != nil {
				if present {
					log.Info("updating option data", "option", name, "value", desiredValue)
				} else {
					log.Info("adding option data", "option", name, "value", desiredValue)
				}
			}
			return &nodes.UpdateOperation{
				Op:    nodes.AddOp, // Add also does replace
				Path:  path,
				Value: desiredValue,
			}
		}
	} else {
		if present {
			if log != nil {
				log.Info("removing option data", "option", name)
			}
			return &nodes.UpdateOperation{
				Op:   nodes.RemoveOp,
				Path: path,
			}
		}
	}
	return nil
}

func sectionUpdateOpts(currentData map[string]interface{}, settings optionsData, basepath string, log logr.Logger) nodes.UpdateOpts {
	var updates nodes.UpdateOpts
	if log != nil && basepath != "" {
		log = log.WithValues("section", basepath[1:])
	}
	for name, desiredValue := range settings {
		path := fmt.Sprintf("%s/%s", basepath, name)
		updateOp := getUpdateOperation(name, currentData, desiredValue, path, log)
		if updateOp != nil {
			updates = append(updates, *updateOp)
		}
	}
	return updates
}

func propertiesUpdateOpts(node *nodes.Node, settings optionsData, log logr.Logger) nodes.UpdateOpts {
	return sectionUpdateOpts(node.Properties, settings, "/properties", log)
}

func instanceInfoUpdateOpts(node *nodes.Node, settings optionsData, log logr.Logger) nodes.UpdateOpts {
	return sectionUpdateOpts(node.InstanceInfo, settings, "/instance_info", log)
}