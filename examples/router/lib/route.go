package lib

import (
	"encoding/json"
	"net/http"
	"reflect"
)

// Action represent a single controller method
type Action struct {
	hasOut bool
	hasErr bool
	ctrler reflect.Value
	method reflect.Method
}

func (r *Action) Execute(w http.ResponseWriter, req *http.Request) {

	var (
		hasOut = r.hasOut
		hasErr = r.hasErr
		method = r.method
		ctrler = r.ctrler
	)

	res := method.Func.Call([]reflect.Value{ctrler, reflect.ValueOf(req), reflect.ValueOf(w)})

	var out any
	var err error
	if hasOut && hasErr {
		if v := res[0]; v.IsValid() {
			out = v.Interface()
		}

		if v := res[1]; !v.IsNil() {
			err = v.Interface().(error)
		}
	} else if hasOut {
		if v := res[0]; v.IsValid() {
			out = v.Interface()
		}
	} else if hasErr {
		if v := res[0]; !v.IsNil() {
			err = v.Interface().(error)
		}
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else if out != nil {
		if str, is := out.(string); is {
			w.Write([]byte(str))
		} else {
			json.NewEncoder(w).Encode(out)
		}

	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}
