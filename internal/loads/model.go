// Package loads implements the AG Grid server-side row model contract for freight loads.
package loads

// Load mirrors the client-side Load model; field names match the seeded JSON/Mongo documents.
type Load struct {
	ID            string  `bson:"id" json:"id"`
	CompanyName   string  `bson:"companyName" json:"companyName"`
	Origin        string  `bson:"origin" json:"origin"`
	Destination   string  `bson:"destination" json:"destination"`
	Weight        float64 `bson:"weight" json:"weight"`
	EquipmentType string  `bson:"equipmentType" json:"equipmentType"`
	Date          string  `bson:"date" json:"date"`
	Price         float64 `bson:"price" json:"price"`
	Distance      float64 `bson:"distance" json:"distance"`
	Status        string  `bson:"status" json:"status"`
}

// stringFields lists the string-typed columns quicksearch matches directly via regex.
// Matches the client's AG Grid quick filter, which searches every column by default.
var stringFields = []string{"id", "companyName", "origin", "destination", "equipmentType", "date", "status"}

// numberFields lists the columns treated as numeric for filtering.
var numberFields = map[string]bool{"weight": true, "price": true, "distance": true}
