package repo

type Repo interface {
	// dsn expected: server=1.1.1.1 port=1009 password=232423
	SetStoreUrl(dsn string) error
	Get(key string) string
	Set(key, val string)
	MapGet(mapKey, subKey string) string
	MapSet(mapKey, subKey, val string)
	MapGetAll(mapKey string) map[string]string
}


