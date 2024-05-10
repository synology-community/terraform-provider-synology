package util

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

func MarshalURL(r any) (url.Values, error) {
	v := reflect.Indirect(reflect.ValueOf(r))
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected type struct, got %T", reflect.TypeOf(r).Name())
	}
	n := v.NumField()
	vT := v.Type()
	ret := url.Values{}
	for i := 0; i < n; i++ {
		urlFieldName := strings.ToLower(vT.Field(i).Name)
		synologyTags := []string{}
		if tags, ok := vT.Field(i).Tag.Lookup("synology"); ok {
			synologyTags = strings.Split(tags, ",")
		}
		if !(vT.Field(i).IsExported() || vT.Field(i).Anonymous || len(synologyTags) > 0) {
			continue
		}
		if len(synologyTags) > 0 {
			urlFieldName = synologyTags[0]
		}

		// get field type
		switch vT.Field(i).Type.Kind() {
		case reflect.String:
			ret.Add(urlFieldName, v.Field(i).String())
		case reflect.Int:
			ret.Add(urlFieldName, strconv.Itoa(int(v.Field(i).Int())))
		case reflect.Bool:
			ret.Add(urlFieldName, strconv.FormatBool(v.Field(i).Bool()))
		case reflect.Slice:
			slice := v.Field(i)
			switch vT.Field(i).Type.Elem().Kind() {
			case reflect.String:
				res := []string{}
				for iSlice := 0; iSlice < slice.Len(); iSlice++ {
					item := slice.Index(iSlice)
					res = append(res, item.String())
				}
				ret.Add(urlFieldName, "[\""+strings.Join(res, "\",\"")+"\"]")
			case reflect.Int:
				res := []string{}
				for iSlice := 0; iSlice < slice.Len(); iSlice++ {
					item := slice.Index(iSlice)
					res = append(res, strconv.Itoa(int(item.Int())))
				}
				ret.Add(urlFieldName, "["+strings.Join(res, ",")+"]")
			}
		case reflect.Struct:
			if !vT.Field(i).Anonymous {
				// support only embedded anonymous structs
				continue
			}
			embStruct := v.Field(i)
			embStructT := v.Field(i).Type()
			for j := 0; j < embStruct.NumField(); j++ {
				synologyTags := strings.Split(embStructT.Field(j).Tag.Get("synology"), ",")
				fieldName := synologyTags[0]
				switch embStruct.Field(j).Kind() {
				case reflect.String:
					ret.Add(fieldName, embStruct.Field(j).String())
				case reflect.Int:
					ret.Add(fieldName, strconv.Itoa(int(embStruct.Field(j).Int())))
				}
			}
		}
	}

	return ret, nil
}
