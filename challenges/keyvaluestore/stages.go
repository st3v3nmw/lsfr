package keyvaluestore

import (
	"fmt"

	"github.com/st3v3nmw/lsfr/internal/registry"
	"github.com/st3v3nmw/lsfr/internal/suite"
)

func init() {
	registry.RegisterStage("key-value-store", 1, Stage1)
}

func Stage1() suite.Suite {
	return *suite.New("Basic SET/GET/DELETE").
		Setup(func(do *suite.Do) error {
			err := do.Run()
			if err != nil {
				return err
			}

			err = do.WaitForPort(8888)
			if err != nil {
				return err
			}

			keys := []string{
				"kenya:capital",
				"uganda:capital",
				"tanzania:capital",
			}
			for _, key := range keys {
				do.HTTP("DELETE", fmt.Sprintf("http://127.0.0.1:8888/kv/%s", key))
			}

			return nil
		}).
		Test("PUT", func(do *suite.Do) {
			do.HTTP("PUT", "http://127.0.0.1:8888/kv/kenya:capital", "Nairobi").
				Got().Status(200)
			do.HTTP("PUT", "http://127.0.0.1:8888/kv/uganda:capital", "Kampala").
				Got().Status(200)
			do.HTTP("PUT", "http://127.0.0.1:8888/kv/tanzania:capital", "Dar es Salaam").
				Got().Status(200)
			do.HTTP("PUT", "http://127.0.0.1:8888/kv/tanzania:capital", "Dodoma").
				Got().Status(200)

			do.HTTP("PUT", "http://127.0.0.1:8888/kv/tanzania:capital").
				Got().Status(400).Body("value cannot be empty\n")
			do.HTTP("PUT", "http://127.0.0.1:8888/kv/", "foo").
				Got().Status(400).Body("key cannot be empty\n")
		}).
		Test("GET", func(do *suite.Do) {
			do.HTTP("GET", "http://127.0.0.1:8888/kv/kenya:capital").
				Got().Status(200).Body("Nairobi")
			do.HTTP("GET", "http://127.0.0.1:8888/kv/tanzania:capital").
				Got().Status(200).Body("Dodoma")

			do.HTTP("GET", "http://127.0.0.1:8888/kv/zanzibar:capital").
				Got().Status(404).Body("key not found\n")
		}).
		Test("DELETE", func(do *suite.Do) {
			do.HTTP("DELETE", "http://127.0.0.1:8888/kv/tanzania:capital").
				Got().Status(200)
			do.HTTP("GET", "http://127.0.0.1:8888/kv/tanzania:capital").
				Got().Status(404).Body("key not found\n")
		})
}
