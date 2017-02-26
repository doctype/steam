package steam

// Filter get InventoryItem and return true if item meet its condition
// false otherwise
type Filter func(*InventoryItem) bool

// IsTradable return Filter for item.Tradable option
func IsTradable(st bool) Filter {
	return func(item *InventoryItem) bool {
		return item.Tradable == st
	}
}
