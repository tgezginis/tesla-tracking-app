package tesla

type TeslaStore struct {
	ID    int
	Label string
}

func GetTeslaStoreByID(id int) TeslaStore {
	if store, ok := teslaStores[id]; ok {
		return store
	}
	return TeslaStore{0, "N/A"}
}

var teslaStores = map[int]TeslaStore{
	0: {0, "N/A"},
	
	436108: {436108, "Dornbirn Mühlebach Pop Up"},
	18438:  {18438, "Graz Kalsdorf"},
	2938:   {2938, "Innsbruck"},
	8730:   {8730, "Klagenfurt"},
	14839:  {14839, "Linz"},
	9340:   {9340, "Wien"},
	18435:  {18435, "Salzburg"},

	
	32061:  {32061, "Awans"},
	14852:  {14852, "Brugge"},

	
	14499:  {14499, "Praha"},

	
	301419: {301419, "Aalborg Storcenter"},
	436102: {436102, "HerningCentret Pop Up"},

	
	438603: {438603, "Espo Pop Up"},
	9118:   {9118, "Turku"},
	26258:  {26258, "Vantaa Petikko"},

	
	446007: {446007, "Boutique éphémère Tesla Nice Cap3000"},
	26558:  {26558, "Centre Tesla Aix-Marseille"},
	30673:  {30673, "Rennes Pacé"},
	26278:  {26278, "Rouen Store"},
	413853: {413853, "Tesla Bayonne"},
	407259: {407259, "Tesla Paris St-Ouen"},
	4004152: {4004152, "Toulon Delivery Hub"},

	
	3693:   {3693, "Augsburg Gersthofen"},
	9194:   {9194, "Berlin - Mall of Berlin"},
	18426:  {18426, "Berlin Reinickendorf"},
	9556:   {9556, "Berlin Schönefeld"},
	16302:  {16302, "Berlin Schönefeld Delivery Hub"},
	25762:  {25762, "Braunschweig Ölper"},
	10512:  {10512, "Bremen Ottersberg"},
	9467:   {9467, "Dortmund Holzwickede"},
	399978: {399978, "Dortmund Innenstadt-Nord"},
	14848:  {14848, "Dresden Kesselsdorf"},
	13495:  {13495, "Duisburg Obermeiderich"},
	14845:  {14845, "Düsseldorf Lierenfeld"},
	439754: {439754, "Flensburg Gallerie Pop Up"},
	20906:  {20906, "Frankfurt Ostend"},
	9093:   {9093, "Freiburg Gundelfingen"},
	28719:  {28719, "Fürth Hardhöhe"},
	28725:  {28725, "Gießen An der Automeile"},
	4225:   {4225, "Hamburg Wandsbek"},
	3951:   {3951, "Hannover Wülfel"},
	438015: {438015, "Heidelberg Altstadt Pop Up"},
	3692:   {3692, "Heilbronn Sontheim"},
	18430:  {18430, "Ingolstadt Oberhaunstadt"},
	3690:   {3690, "Karlsruhe Rintheim"},
	9095:   {9095, "Kiel Gettorf"},
	18422:  {18422, "Koblenz Mülheim-Kärlich"},
	2841:   {2841, "Köln Mülheim"},
	20823:  {20823, "Magdeburg Großer Silberberg"},
	1501:   {1501, "Mannheim Friedrichsfeld"},
	2614:   {2614, "München Freiham"},
	18423:  {18423, "München Parsdorf"},
	15929:  {15929, "Neu-Ulm Schwaighofen"},
	1250:   {1250, "Nürnberg St. Jobst"},
	439780: {439780, "Rosenheim Innenstadt Pop Up"},
	9098:   {9098, "Rostock Nienhagen"},
	26292:  {26292, "Saarbrücken Brebach-Fechingen"},
	27044:  {27044, "Stuttgart Holzgerlingen Sales, Used Car & Delivery Center"},
	28717:  {28717, "Stuttgart Weinstadt"},

	
	425125: {425125, "Tesla Armada AVM"},
	445210: {445210, "Tesla Ferko Line"},
	449153: {449153, "Tesla Istanbul Meydan AVM"},
	451952: {451952, "Tesla İstinyePark İzmir"},
	410805: {410805, "Tesla Ankara Delivery Hub"},
	442359: {442359, "Tesla Delivery Istanbul"},
	460569: {460569, "Tesla Delivery Gaziemir Izmir"},
} 