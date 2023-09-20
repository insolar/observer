// +build !node

package configuration

type CollectorBinance struct {
	Log Log
	DB  DB
}

type CollectorCoinMarketCap struct {
	Log Log
	DB  DB
}

type StatsCollector struct {
	Log Log
	DB  DB
}

func (StatsCollector) Default() StatsCollector {
	return StatsCollector{
		DB:  Observer{}.Default().DB,
		Log: Observer{}.Default().Log,
	}
}

func Configurations() map[string]interface{} {
	cfgs := make(map[string]interface{})
	cfgs["observerapi.yaml"] = APIExtended{}.Default()
	cfgs["observer.yaml"] = Observer{}.Default()
	cfgs["migrate.yaml"] = Migrate{}.Default()
	cfgs["stats-collector.yaml"] = StatsCollector{}.Default()

	return cfgs
}
