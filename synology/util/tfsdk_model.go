package util

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// func ToDynamicValue(t attr.Value) (attr.Value, error) {

// }

func GetType(r any) (attr.Type, error) {
	var v reflect.Value

	if reflect.TypeOf(r).Kind() == reflect.Ptr {
		v = reflect.Indirect(reflect.ValueOf(r))
	} else {
		v = reflect.ValueOf(r)
	}

	vk := v.Kind()
	switch vk {
	case reflect.Struct:
		embAttrTypes, err := structType(v)
		if err != nil {
			return nil, err
		}
		return embAttrTypes, nil
	case reflect.Slice:
		embAttrType, err := sliceType(v.Index(0))
		if err != nil {
			return nil, err
		}
		return embAttrType, nil
	case reflect.Map:
		embAttrType, err := mapType(v)
		if err != nil {
			return nil, err
		}
		return embAttrType, nil
	}
	return nil, fmt.Errorf("unsupported type %s", vk)
}

func mapType(v reflect.Value) (attr.Type, error) {
	attrTypes := map[string]attr.Type{}
	if reflect.TypeOf(v).Kind() == reflect.Ptr || reflect.TypeOf(v).Kind() == reflect.Interface {
		v = v.Elem()
	}

	iter := v.MapRange()
	for iter.Next() {
		k := iter.Key()
		iv := iter.Value()

		vT := iv.Type()
		vK := vT.Kind()

		if (vK == reflect.Ptr) || (vK == reflect.Interface) {
			iv = iv.Elem()
			vT = iv.Type()
			vK = vT.Kind()
		}

		switch vK {
		case reflect.String:
			attrTypes[k.String()] = types.StringType
		case reflect.Int64, reflect.Int, reflect.Int32, reflect.Int16, reflect.Int8:
			attrTypes[k.String()] = types.Int64Type
		case reflect.Uint,
			reflect.Uint32,
			reflect.Uint16,
			reflect.Uint8,
			reflect.Uintptr,
			reflect.Uint64:
			attrTypes[k.String()] = types.Int64Type
		case reflect.Float32, reflect.Float64:
			attrTypes[k.String()] = types.Float64Type
		case reflect.Bool:
			attrTypes[k.String()] = types.BoolType
		case reflect.Struct:
			cv := reflect.New(v.Type()).Elem()
			embAttrType, err := structType(cv)
			if err != nil {
				return nil, err
			}
			attrTypes[k.String()] = embAttrType
		case reflect.Slice, reflect.Array:
			embAttrType, err := sliceType(v)
			if err != nil {
				return nil, err
			}
			attrTypes[k.String()] = embAttrType
		default:
			attrTypes[k.String()] = types.DynamicType
		}
	}
	return types.ObjectType{}.WithAttributeTypes(attrTypes), nil
}

func sliceType(v reflect.Value) (attr.Type, error) {
	vT := v.Type()
	switch vT.Elem().Kind() {
	case reflect.String:
		return types.ListType{}.WithElementType(types.StringType), nil
	case reflect.Int64, reflect.Int, reflect.Int32, reflect.Int16, reflect.Int8:
		return types.ListType{}.WithElementType(types.Int64Type), nil
	case reflect.Uint,
		reflect.Uint32,
		reflect.Uint16,
		reflect.Uint8,
		reflect.Uintptr,
		reflect.Uint64:
		return types.ListType{}.WithElementType(types.Int64Type), nil
	case reflect.Float32, reflect.Float64:
		return types.ListType{}.WithElementType(types.Float64Type), nil
	case reflect.Bool:
		return types.ListType{}.WithElementType(types.BoolType), nil
	case reflect.Struct:
		cv := reflect.New(vT.Elem()).Elem()
		embAttrType, err := structType(cv)
		if err != nil {
			return nil, err
		}
		return types.ListType{}.WithElementType(embAttrType), nil
	default:
		return types.ListType{}.WithElementType(types.DynamicType), nil
	}
}

func structType(v reflect.Value) (attr.Type, error) {
	attrTypes := map[string]attr.Type{}
	n := v.NumField()
	vT := v.Type()

	for i := 0; i < n; i++ {
		field := vT.Field(i)
		fieldType := field.Type

		attrFieldName := strings.ToLower(field.Name)
		jsonTags := []string{}
		if tags, ok := field.Tag.Lookup("json"); ok {
			jsonTags = strings.Split(tags, ",")
		}
		if !field.IsExported() && !field.Anonymous && len(jsonTags) <= 0 {
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
			attrTypes[attrFieldName] = types.StringType
		case reflect.Int64:
			attrTypes[attrFieldName] = types.Int64Type
		case reflect.Int:
			attrTypes[attrFieldName] = types.Int64Type
		case reflect.Bool:
			attrTypes[attrFieldName] = types.BoolType
		case reflect.Slice:
			embAttrType, err := sliceType(v.Field(i))
			if err != nil {
				return nil, err
			}
			attrTypes[attrFieldName] = embAttrType
		case reflect.Struct:
			embAttrType, err := structType(v.Field(i))
			if err != nil {
				return nil, err
			}
			attrTypes[attrFieldName] = embAttrType
		}
	}
	return types.ObjectType{}.WithAttributeTypes(attrTypes), nil
}

func sliceValue(v reflect.Value) (attr.Value, error) {
	attrValues := []attr.Value{}
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i)
		switch item.Kind() {
		case reflect.String:
			attrValues = append(attrValues, types.StringValue(item.String()))
		case reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
			attrValues = append(attrValues, types.Int64Value(item.Int()))
		case reflect.Uint,
			reflect.Uint32,
			reflect.Uint16,
			reflect.Uint8,
			reflect.Uintptr,
			reflect.Uint64:
			attrValues = append(attrValues, types.Int64Value(int64(item.Uint())))
		case reflect.Float32, reflect.Float64:
			attrValues = append(attrValues, types.Float64Value(item.Float()))
		case reflect.Struct:

			embAttrValues, err := structValue(item)
			if err != nil {
				return nil, err
			}
			attrValues = append(attrValues, embAttrValues)
		}
	}
	attrType, err := sliceType(v)
	if err != nil {
		return nil, err
	}
	if listType, ok := attrType.(types.ListType); ok {
		return types.ListValueMust(listType.ElementType(), attrValues), nil
	} else {
		return nil, fmt.Errorf("unsupported type %T", attrType)
	}
}

func mapValue(v reflect.Value) (attr.Value, error) {
	if reflect.TypeOf(v).Kind() == reflect.Ptr || reflect.TypeOf(v).Kind() == reflect.Interface {
		v = v.Elem()
	}

	attrValues := map[string]attr.Value{}
	iter := v.MapRange()
	for iter.Next() {
		k := iter.Key()
		iv := iter.Value()

		vT := iv.Type()
		vK := vT.Kind()

		if vK == reflect.Ptr || vK == reflect.Interface {
			iv = iv.Elem()
			vT = iv.Type()
			vK = vT.Kind()
		}

		switch vK {
		case reflect.String:
			attrValues[k.String()] = types.StringValue(iv.String())
		case reflect.Int64, reflect.Int, reflect.Int32, reflect.Int16, reflect.Int8:
			attrValues[k.String()] = types.Int64Value(iv.Int())
		case reflect.Uint,
			reflect.Uint32,
			reflect.Uint16,
			reflect.Uint8,
			reflect.Uintptr,
			reflect.Uint64:
			attrValues[k.String()] = types.Int64Value(int64(iv.Uint()))
		case reflect.Float32, reflect.Float64:
			attrValues[k.String()] = types.Float64Value(iv.Float())
		case reflect.Bool:
			attrValues[k.String()] = types.BoolValue(iv.Bool())
		case reflect.Struct:
			embAttrValues, err := structValue(iv)
			if err != nil {
				return nil, err
			}
			attrValues[k.String()] = embAttrValues
		case reflect.Array, reflect.Slice:
			embAttrValues, err := sliceValue(iv)
			if err != nil {
				return nil, err
			}
			attrValues[k.String()] = embAttrValues
		}
	}
	attrType, err := mapType(v)
	if err != nil {
		return nil, err
	}
	if objectType, ok := attrType.(types.ObjectType); ok {
		return types.ObjectValueMust(objectType.AttributeTypes(), attrValues), nil
	} else {
		return nil, fmt.Errorf("unsupported type %T", attrType)
	}
}

func structValue(v reflect.Value) (attr.Value, error) {
	attrValues := map[string]attr.Value{}
	n := v.NumField()
	vT := v.Type()

	for i := 0; i < n; i++ {
		field := vT.Field(i)
		fieldType := field.Type

		attrFieldName := strings.ToLower(field.Name)
		jsonTags := []string{}
		if tags, ok := field.Tag.Lookup("json"); ok {
			jsonTags = strings.Split(tags, ",")
		}
		if !field.IsExported() && !field.Anonymous && len(jsonTags) <= 0 {
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
			attrValues[attrFieldName] = types.StringValue(v.Field(i).String())
		case reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
			attrValues[attrFieldName] = types.Int64Value(v.Field(i).Int())
		case reflect.Uint,
			reflect.Uint32,
			reflect.Uint16,
			reflect.Uint8,
			reflect.Uintptr,
			reflect.Uint64:
			attrValues[attrFieldName] = types.Int64Value(int64(v.Field(i).Uint()))
		case reflect.Float32, reflect.Float64:
			attrValues[attrFieldName] = types.Float64Value(v.Field(i).Float())
		case reflect.Bool:
			attrValues[attrFieldName] = types.BoolValue(v.Field(i).Bool())
		case reflect.Slice, reflect.Array:
			attrValue, err := sliceValue(v.Field(i))
			if err != nil {
				return nil, err
			}
			attrValues[attrFieldName] = attrValue
		case reflect.Struct:
			attrValue, err := structValue(v.Field(i))
			if err != nil {
				return nil, err
			}
			attrValues[attrFieldName] = attrValue
		}
	}

	attrType, err := structType(v)
	if err != nil {
		return nil, err
	}
	objectType, ok := attrType.(types.ObjectType)
	if !ok {
		return nil, fmt.Errorf("unsupported type %T", attrType)
	}
	return types.ObjectValueMust(objectType.AttributeTypes(), attrValues), nil
}

func GetValue(r any) (attr.Value, error) {
	var v reflect.Value

	if reflect.TypeOf(r).Kind() == reflect.Ptr {
		v = reflect.Indirect(reflect.ValueOf(r))
	} else {
		v = reflect.ValueOf(r)
	}

	if !v.IsValid() {
		vt := reflect.TypeOf(v)
		if vt.Kind() == reflect.Ptr {
			v = reflect.Indirect(reflect.ValueOf(v))
		}
	}

	vk := v.Kind()
	switch vk {
	case reflect.Struct:
		return structValue(v)
	case reflect.Slice:
		return sliceValue(v.Index(0))
	case reflect.Map:
		return mapValue(v)
	}
	return nil, fmt.Errorf("unsupported type %s", vk)
}

// func GetValue(r any) (attr.Value, error) {
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
