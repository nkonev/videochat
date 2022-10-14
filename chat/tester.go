package main

import (
	"fmt"
	"nkonev.name/chat/handlers/dto"
	"nkonev.name/chat/services"
	"reflect"
)

var typeRegistry = make(map[string]reflect.Type)

func main() {
	cd := &dto.DisplayMessageDto{
		Id: 123,
	}
	aDto := services.MessageNotify{
		Type:                "eventTypeCreatedPepyake",
		MessageNotification: cd,
	}

	strName := addToRegistry(aDto)
	fmt.Printf("> %v\n", strName)

	notify := makeInstance("services.MessageNotify").(services.MessageNotify)
	notify.Type = "sty"
	fmt.Printf(">> %v\n", notify)
}

func addToRegistry(aDto interface{}) (strName string) {
	strName = fmt.Sprintf("%T", aDto)
	typeRegistry[strName] = reflect.TypeOf(aDto)
	return
}

func makeInstance(name string) interface{} {
	v := reflect.New(typeRegistry[name]).Elem()
	return v.Interface()
}
