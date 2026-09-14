package provider

// SetBases points every provider endpoint at one base URL. It exists so tests
// (and local experiments) can redirect providers to a fixture server; it is not
// part of any external contract (this is an internal package).
func SetBases(base string) {
	coingeckoBase = base
	frankfurterBase = base
	yahooBase = base + "/chart"
	cboeBase = base
	edgarData = base
	edgarWWW = base
	treasuryBase = base
	fiscalDataBase = base
	cftcBase = base + "/cftc.json"
	kalshiBase = base
	fredBase = base
	twseBase = base
	vastBase = base
}
