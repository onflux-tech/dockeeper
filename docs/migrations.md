# Migrations — Dockeeper

## Quando usar

Use para criar ou alterar collections, campos, índices, relações e regras do PocketBase.

## Checklist

1. Crie `migrations/NN_nome.go` com o próximo número de dois dígitos. O pacote é `migrations`.
2. Registre com `m.Register(func(app core.App) error { ... }, down)`. O segundo argumento é o rollback; use `nil` quando não houver.
3. Para collection nova: `core.NewBaseCollection("nome")`, `Fields.Add(...)`, `AddIndex(...)` e `app.Save(collection)`.
4. Para collection existente: `app.FindCollectionByNameOrId("nome")`, altere `Fields`/índices e salve.
5. Adicione a constante `Coll...` em `internal/shared/constants/collections.go`.
6. Se o dado cruza manager/agent ou é lido por mais de uma camada, adicione a struct em `internal/shared/models/models.go`.
7. Confirme o blank import do pacote `migrations` em `cmd/manager/main.go`. Ele é único para o pacote; não adicione imports por arquivo.
8. Rode `make test` e `make verify-fast`.

## Regras

- Nunca edite uma migration já publicada. Crie a próxima com numeração sequencial.
- `migrations/migrations.go` é só o pacote; não coloque imports lá.
- Relações entre collections usam `getCollectionID(app, "nome")` quando a ordem de criação exigir.
- Índices nomeados: `idx_<collection>_<campo>`.
- O PocketBase aplica migrations no boot do manager; reinicie após criar ou alterar uma.
- Se a collection não aparecer em runtime, confira o blank import e os logs de migration.
