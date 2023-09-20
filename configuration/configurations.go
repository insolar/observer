// +build node

package configuration

func Configurations() map[string]interface{} {
	cfgs := make(map[string]interface{})
	cfgs["observerapi.yaml"] = API{}.Default()
	cfgs["observer.yaml"] = Observer{}.Default()
	cfgs["migrate.yaml"] = Migrate{}.Default()

	return cfgs
}
