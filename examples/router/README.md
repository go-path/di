# go-path/di router example

Simple example of how you can use **go-path/di** to structure a project's routes, reducing coupling between application modules.

The `controller` directory has the controllers that are automatically initialized by the `Router`.


`lib.Router` obtains from `Container` all instances that have the `Path() string` method. After that, it uses reflection to obtain the methods with the pattern `$MethodName(r http.Request, w.HttpWriter) [response, error]` and maps the route automatically.


`lib.Server` initializes the http server

Routes

- http://localhost:8081/
- http://localhost:8081/ping
- http://localhost:8081/star-wars
- http://localhost:8081/star-wars/people?id=1
- http://localhost:8081/star-wars/planets?id=1
- http://localhost:8081/star-wars/starship?id=1



