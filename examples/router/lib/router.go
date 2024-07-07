package lib

import (
	"net/http"
	"path"
	"reflect"
	"strings"

	"github.com/go-path/di"
)

// Router initialize controllers
type Router struct {
	routes      map[string]*Action
	Controllers []Controller `inject:""`
}

// ServeHTTP Router is a http.Handler (see Server)
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if route, exists := r.routes[req.URL.Path]; exists {
		route.Execute(w, req)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// register controllers routers, use reflection to obtain the methods
func (r *Router) Initialize() {

	r.routes = map[string]*Action{}

	for _, controller := range r.Controllers {

		ctrlValue := reflect.ValueOf(controller)
		ctrlType := ctrlValue.Type()
		ctrlPath := controller.Path()

		if strings.IndexByte(ctrlPath, '/') != 0 {
			ctrlPath = "/" + ctrlPath
		}

		numMethod := ctrlType.NumMethod()
		for i := 0; i < numMethod; i++ {

			// `$MethodName(r *http.Request, w http.ResponseWriter) [response, error]
			method := ctrlType.Method(i)

			if !method.IsExported() {
				continue
			}

			if method.Type.NumIn() != 3 {
				// receiver, r *http.Request, w http.ResponseWriter
				continue
			}

			if !method.Type.In(1).AssignableTo(reflect.TypeOf((*http.Request)(nil))) {
				// not *http.Request
				continue
			}

			if !method.Type.In(2).Implements(reflect.TypeOf((*http.ResponseWriter)(nil)).Elem()) {
				// not http.ResponseWriter
				continue
			}

			hasOut := false
			hasErr := false
			numOut := method.Type.NumOut()
			if numOut > 0 {
				hasOut = true

				out1 := method.Type.Out(0)
				if out1.Implements(reflect.TypeOf((*error)(nil)).Elem()) {
					hasErr = true
				} else {
					hasOut = true
				}

				if numOut > 1 {
					if hasErr {
						continue
					}

					out2 := method.Type.Out(1)
					if !out2.Implements(reflect.TypeOf((*error)(nil)).Elem()) {
						continue
					}

					hasOut = true
					hasErr = true
				}
			}

			mRoute := strings.ToLower(method.Name)
			if mRoute == "index" {
				mRoute = ""
			}
			route := path.Join(ctrlPath, mRoute)

			r.routes[route] = &Action{
				hasOut: hasOut,
				hasErr: hasErr,
				method: method,
				ctrler: ctrlValue,
			}
		}
	}
}

func init() {
	// register as startup component, injecting dependencies
	// executes before server startup
	di.Injected[*Router](di.Startup(100))
}
