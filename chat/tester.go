package main

import (
	"fmt"
	"go/types"
	"nkonev.name/chat/handlers/dto"
	"nkonev.name/chat/services"
	"reflect"
)

func main() {
	cd := &dto.DisplayMessageDto{
		Id: 123,
	}
	fmt.Printf("The type of name is: %T\n", cd)
	fmt.Println(reflect.TypeOf(cd)) // main.rectangle
	//
	aDto := services.MessageNotify{
		Type:                "eventTypeCreatedPepyake",
		MessageNotification: cd,
	}

	aTypeStr := reflect.TypeOf(aDto).String()
	// 	newPackage := types.NewPackage("nkonev.name/chat", "services")
	newPackage := types.NewPackage("nkonev.name/chat", "utils")
	id := types.Id(newPackage, aTypeStr)

	fmt.Printf("A type is str -> %v , id -> %v\n", aTypeStr, id)

	intType := types.NewType()
	bindTo := reflect.New(intType)

	fmt.Printf("A reconstructed type is %v %v\n", intType, bindTo)

}
