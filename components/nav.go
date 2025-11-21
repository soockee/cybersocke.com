package components

// GetNavItems returns navigation items based on authentication/role context.
// Future extension could accept roles slice; for now a simple authed flag.
func GetNavItems(authed bool) []NavItem {
	items := []NavItem{
		NewNavItem("Home", "/"),
		NewNavItem("Graph", "/graph"),
	}
	if authed {
		items = append(items, NewNavItem("Admin", "/admin"))
	}
	return items
}
