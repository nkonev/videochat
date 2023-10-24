package type_registry

import (
	"nkonev.name/event/dto"
	"nkonev.name/event/utils"
	"reflect"
)

type TypeRegistryInstance struct {
	typeRegistry map[string]reflect.Type
}

func NewTypeRegistryInstance() *TypeRegistryInstance {
	var typeRegistry = make(map[string]reflect.Type)
	var res = &TypeRegistryInstance{
		typeRegistry: typeRegistry,
	}

	res.AddToRegistryIfNeed(dto.ChatEvent{})
	res.AddToRegistryIfNeed(dto.UserEvent{})
	res.AddToRegistryIfNeed([]dto.UserOnline{})
	res.AddToRegistryIfNeed(dto.GlobalEvent{})
	return res
}

func (tr *TypeRegistryInstance) AddToRegistry(aDto interface{}) (strName string) {
	strName = utils.GetType(aDto)
	tr.typeRegistry[strName] = reflect.TypeOf(aDto)
	return
}

func (tr *TypeRegistryInstance) AddToRegistryIfNeed(aDto interface{}) string {
	strName := utils.GetType(aDto)
	_, ok := tr.typeRegistry[strName]
	if !ok {
		return tr.AddToRegistry(aDto)
	} else {
		return strName
	}
}

func (tr *TypeRegistryInstance) MakeInstance(name string) interface{} {
	v := reflect.New(tr.typeRegistry[name]).Elem()
	return v.Interface()
}

func (tr *TypeRegistryInstance) GetType(aDto interface{}) string {
	strName := utils.GetType(aDto)
	return strName
}

func (tr *TypeRegistryInstance) HasType(strName string) bool {
	_, ok := tr.typeRegistry[strName]
	return ok
}
