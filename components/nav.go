package components

import "strings"

// GetNavItems returns navigation items based on authentication/role context.
// Future extension could accept roles slice; for now a simple authed flag.
func GetNavItems(authed bool) []NavItem {
	items := []NavItem{
		NewNavItem("Posts", "/posts"),
		NewNavItem("Explore", "/graph"),
	}
	if authed {
		items = append(items, NewNavItem("Admin", "/admin"))
	}
	return items
}

// NavUser summarizes the authenticated visitor for navbar display.
type NavUser struct {
	UID        string
	Name       string
	Email      string
	PictureURL string
}

// NavBarProps bundles navigation items with authentication metadata for rendering.
type NavBarProps struct {
	Items  []NavItem
	Authed bool
	User   *NavUser
}

// BuildNavProps constructs NavBarProps from a simple auth flag and optional user info.
func BuildNavProps(authed bool, user *NavUser) NavBarProps {
	return NavBarProps{Items: GetNavItems(authed), Authed: authed, User: user}
}

// Initials returns up to two uppercase initials for avatar fallbacks.
func (u *NavUser) Initials() string {
	if u == nil {
		return ""
	}
	name := strings.TrimSpace(u.Name)
	if name != "" {
		parts := strings.Fields(name)
		if len(parts) == 1 {
			return strings.ToUpper(string([]rune(parts[0])[0]))
		}
		return strings.ToUpper(string([]rune(parts[0])[0])) + strings.ToUpper(string([]rune(parts[len(parts)-1])[0]))
	}
	if u.Email != "" {
		r := []rune(u.Email)
		return strings.ToUpper(string(r[0]))
	}
	if u.UID != "" {
		r := []rune(u.UID)
		return strings.ToUpper(string(r[0]))
	}
	return ""
}
