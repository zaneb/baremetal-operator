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

type nodeUpdater struct {
	node    *nodes.Node
	log     logr.Logger
	updates nodes.UpdateOpts
}

type nodeUpdaterInterface interface {
	Root() nodeSectionUpdater
	InstanceInfo() nodeSectionUpdater
	Properties() nodeSectionUpdater

	Updates() nodes.UpdateOpts
}

type nodeSectionUpdater interface {
	nodeUpdaterInterface
	SetOpt(name string, desiredValue interface{}) nodeSectionUpdater
	ClearOpt(name string) nodeSectionUpdater
}

func updateNode(node *nodes.Node, log logr.Logger) *nodeUpdater {
	return &nodeUpdater{
		node: node,
		log:  log,
	}
}

func (nu *nodeUpdater) Updates() nodes.UpdateOpts {
	return nu.updates
}

func (nu *nodeUpdater) section(sectionName string, data map[string]interface{}) nodeSectionUpdater {
	logger := nu.log
	section := sectionName
	if sectionName != "" {
		if logger != nil {
			logger = logger.WithValues("section", sectionName)
		}
		section = "/" + sectionName
	}
	return &sectionUpdater{
		nodeUpdater: nu,
		data:        data,
		section:     section,
		log:         logger,
	}
}

func (nu *nodeUpdater) Root() nodeSectionUpdater {
	return &rootUpdater{nu}
}

func (nu *nodeUpdater) InstanceInfo() nodeSectionUpdater {
	return nu.section("instance_info", nu.node.InstanceInfo)
}

func (nu *nodeUpdater) Properties() nodeSectionUpdater {
	return nu.section("properties", nu.node.Properties)
}

type rootUpdater struct {
	*nodeUpdater
}

func (ru *rootUpdater) optValue(name string) interface{} {
	nodeType := reflect.TypeOf(*ru.node)
	numFields := nodeType.NumField()
	for i := 0; i < numFields; i++ {
		f := nodeType.Field(i)
		if n, ok := f.Tag.Lookup("json"); ok && n == name {
			value := reflect.ValueOf(*ru.node).FieldByName(f.Name)
			if !value.IsValid() {
				return nil
			}
			return value.Interface()
		}
	}
	return nil
}

func (ru *rootUpdater) sectionWithOpt(name string) nodeSectionUpdater {
	currentData := map[string]interface{}{}
	if v := ru.optValue(name); v != nil {
		currentData[name] = v
	}
	return ru.section("", currentData)
}

func (ru *rootUpdater) SetOpt(name string, desiredValue interface{}) nodeSectionUpdater {
	ru.sectionWithOpt(name).SetOpt(name, desiredValue)
	return ru
}

func (ru *rootUpdater) ClearOpt(name string) nodeSectionUpdater {
	ru.sectionWithOpt(name).ClearOpt(name)
	return ru
}

type sectionUpdater struct {
	*nodeUpdater
	data    map[string]interface{}
	section string
	log     logr.Logger
}

func (su *sectionUpdater) path(option string) string {
	return fmt.Sprintf("%s/%s", su.section, option)
}

func (su *sectionUpdater) SetOpt(name string, desiredValue interface{}) nodeSectionUpdater {
	current, present := su.data[name]
	desiredValue = deref(desiredValue)
	if !(present && optionValueEqual(deref(current), desiredValue)) {
		if su.log != nil {
			su.log = su.log.WithValues("option", name, "value", desiredValue)
			if present {
				su.log.Info("updating option data")
			} else {
				su.log.Info("adding option data")
			}
		}
		su.updates = append(su.updates,
			nodes.UpdateOperation{
				Op:    nodes.AddOp, // Add also does replace
				Path:  su.path(name),
				Value: desiredValue,
			})
	}
	return su
}

func (su *sectionUpdater) ClearOpt(name string) nodeSectionUpdater {
	_, present := su.data[name]
	if present {
		if su.log != nil {
			su.log.Info("removing option data", "option", name)
		}
		su.updates = append(su.updates,
			nodes.UpdateOperation{
				Op:   nodes.RemoveOp,
				Path: su.path(name),
			})
	}
	return su
}
