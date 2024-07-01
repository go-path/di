# go-path/di router example

Exemplo simples de como pode usar o **go-path/di** para estruturar as rotas de um projeto, diminuindo o acoplamento entre os módulos da aplicação.

O diretório `controller` possui as controllers que são inicializadas automaticamente pelo `Router`.


O `lib.Router` obtém do `Container` todas as instancias que possui o método `Path() string`. Após isso, faz uso de reflection para obter os métodos com o padão `$MethodName(r http.Request, w.HttpWriter) [response, error]` e faz o mapeamento da rota automaticamente.


O `lib.Server` faz a inicialização do servidor http, utilizando  



