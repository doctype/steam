package steam

// Filter get InventoryItem and return true if item meet its condition
// false otherwise
type Filter func(*InventoryItem) bool

// IsTradable return Filter for item.Tradable option
func IsTradable(cond bool) Filter {
	return func(item *InventoryItem) bool {
		return item.Desc.Tradable == cond
	}
}

// IsSouvenir filters souvenir items
func IsSouvenir(cond bool) Filter {
	return func(item *InventoryItem) bool {
		for _, tag := range item.Desc.Tags {
			if tag.Category == "Quality" && tag.InternalName == "tournament" {
				return cond
			}
		}

		return !cond
	}
}
