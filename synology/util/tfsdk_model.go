package util

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func GetType(r interface{}) (map[string]attr.Type, error) {
	result := map[string]attr.Type{}
	var v reflect.Value

	if reflect.TypeOf(r).Kind() == reflect.Ptr {
		v = reflect.Indirect(reflect.ValueOf(r))
	} else {
		v = reflect.ValueOf(r)
	}

	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected type struct, got %T", v.Type().Name())
	}
	n := v.NumField()
	vT := v.Type()

	for i := 0; i < n; i++ {
		field := vT.Field(i)
		fieldType := field.Type

		// if fieldType.Kind() == reflect.Ptr {
		// 	if v.Field(i).IsNil() {
		// 		continue
		// 	}
		// 	fieldType = fieldType.Elem()
		// }

		attrFieldName := strings.ToLower(field.Name)
		jsonTags := []string{}
		if tags, ok := field.Tag.Lookup("json"); ok {
			jsonTags = strings.Split(tags, ",")
		}
		if !(field.IsExported() || field.Anonymous || len(jsonTags) > 0) {
			continue
		}
		if len(jsonTags) > 0 {
			attrFieldName = jsonTags[0]
			if attrFieldName == "-" {
				continue
			}
		}

		attrKind := fieldType.Kind()

		// get field type
		switch attrKind {
		case reflect.String:
			result[attrFieldName] = types.StringType
		case reflect.Int64:
			result[attrFieldName] = types.Int64Type
		case reflect.Int:
			result[attrFieldName] = types.NumberType
		case reflect.Bool:
			result[attrFieldName] = types.BoolType
		case reflect.Slice:
			switch fieldType.Elem().Kind() {
			case reflect.String:
				result[attrFieldName] = types.ListType{}.WithElementType(types.StringType)
			case reflect.Int64:
				result[attrFieldName] = types.ListType{}.WithElementType(types.Int64Type)
			case reflect.Int:
				result[attrFieldName] = types.ListType{}.WithElementType(types.NumberType)
			case reflect.Struct:
				cv := reflect.New(fieldType.Elem()).Elem()
				c := cv.Interface()
				embAttrTypes, err := GetType(c)
				if err != nil {
					return nil, err
				}
				result[attrFieldName] = types.ListType{}.WithElementType(types.ObjectType{}.WithAttributeTypes(embAttrTypes))
			}
		case reflect.Struct:
			embAttrTypes, err := GetType(v.Field(i).Interface())

			if err != nil {
				return nil, err
			}

			result[attrFieldName] = types.ObjectType{}.WithAttributeTypes(embAttrTypes)
		}
	}

	return result, nil
}

// func GetValue(r interface{}) (attr.Value, error) {
// 	var result attr.Value
// 	var valueResult attr.Value

// 	var v reflect.Value

// 	if reflect.TypeOf(r).Kind() == reflect.Ptr {
// 		v = reflect.Indirect(reflect.ValueOf(r))
// 	} else {
// 		v = reflect.ValueOf(r)
// 	}
// 	if v.Kind() != reflect.Struct {
// 		return nil, fmt.Errorf("expected type struct, got %T", reflect.TypeOf(r).Name())
// 	}
// 	n := v.NumField()
// 	vT := v.Type()

// 	for i := 0; i < n; i++ {
// 		field := vT.Field(i)
// 		fieldType := field.Type

// 		// if fieldType.Kind() == reflect.Ptr {
// 		// 	if v.Field(i).IsNil() {
// 		// 		continue
// 		// 	}
// 		// 	fieldType = fieldType.Elem()
// 		// }

// 		attrFieldName := strings.ToLower(field.Name)
// 		attrKindName := "field"
// 		jsonTags, kindTags := []string{}, []string{}
// 		if tags, ok := field.Tag.Lookup("form"); ok {
// 			jsonTags = strings.Split(tags, ",")
// 		}
// 		if tags, ok := field.Tag.Lookup("kind"); ok {
// 			kindTags = strings.Split(tags, ",")
// 		}
// 		if !(field.IsExported() || field.Anonymous || len(jsonTags) > 0) {
// 			continue
// 		}
// 		if len(jsonTags) > 0 {
// 			attrFieldName = jsonTags[0]
// 			if attrFieldName == "-" {
// 				continue
// 			}
// 		}
// 		if len(kindTags) > 0 {
// 			attrKindName = kindTags[0]
// 		}

// 		attrKind := fieldType.Kind()

// 		if field.Name == "File" {
// 			attrKind = reflect.Struct
// 		}

// 		// get field type
// 		switch attrKind {
// 		case reflect.String:
// 			if err := w.WriteField(attrFieldName, v.Field(i).String()); err != nil {
// 				return nil, err
// 			}
// 		case reflect.Int:
// 			if err := w.WriteField(attrFieldName, strconv.Itoa(int(v.Field(i).Int()))); err != nil {
// 				return nil, err
// 			}
// 		case reflect.Bool:
// 			result[attrFieldName] = types.BoolType
// 			if err := w.WriteField(attrFieldName, strconv.FormatBool(v.Field(i).Bool())); err != nil {
// 				return nil, err
// 			}
// 		case reflect.Slice:
// 			slice := v.Field(i)
// 			switch fieldType.Elem().Kind() {
// 			case reflect.String:
// 				res := []string{}
// 				for iSlice := 0; iSlice < slice.Len(); iSlice++ {
// 					item := slice.Index(iSlice)
// 					res = append(res, item.String())
// 				}
// 				result[attrFieldName] = types.StringType
// 				if err := w.WriteField(attrFieldName, "[\""+strings.Join(res, "\",\"")+"\"]"); err != nil {
// 					return nil, err
// 				}
// 			case reflect.Int:
// 				res := []string{}
// 				for iSlice := 0; iSlice < slice.Len(); iSlice++ {
// 					item := slice.Index(iSlice)
// 					res = append(res, strconv.Itoa(int(item.Int())))
// 				}
// 				if err := w.WriteField(attrFieldName, "["+strings.Join(res, ",")+"]"); err != nil {
// 					return nil, err
// 				}
// 			}
// 		case reflect.Struct:
// 			// if !field.Anonymous {
// 			// 	// support only embedded anonymous structs
// 			// 	continue
// 			// }

// 			var fileName string
// 			var fileReader io.Reader

// 			embStruct := v.Field(i)
// 			embStructT := v.Field(i).Type()
// 			for j := 0; j < embStruct.NumField(); j++ {
// 				jsonTags := strings.Split(embStructT.Field(j).Tag.Get("form"), ",")
// 				fieldName := jsonTags[0]
// 				switch embStruct.Field(j).Kind() {
// 				case reflect.String:
// 					switch attrKindName {
// 					case "file":
// 						switch fieldName {
// 						case "content":
// 							fileReader = strings.NewReader(embStruct.Field(j).String())
// 						case "name":
// 							fileName = embStruct.Field(j).String()
// 						default:
// 							if err := w.WriteField(fieldName, embStruct.Field(j).String()); err != nil {
// 								return nil, err
// 							}
// 						}
// 					default:
// 						if err := w.WriteField(fieldName, embStruct.Field(j).String()); err != nil {
// 							return nil, err
// 						}
// 					}
// 				case reflect.Int:
// 					if err := w.WriteField(fieldName, strconv.Itoa(int(embStruct.Field(j).Int()))); err != nil {
// 						return nil, err
// 					}
// 				}
// 			}

// 			if attrKindName == "file" {
// 				if fw, err := w.CreateFormFile(attrFieldName, fileName); err != nil {
// 					return nil, err
// 				} else {

// 					if size, err := io.Copy(fw, fileReader); err != nil {
// 						return nil, err
// 					} else {
// 						fileSize = size
// 					}

// 				}
// 			}
// 		}
// 	}
// }
