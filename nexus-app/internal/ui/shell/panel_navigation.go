package shell

import "strings"

func (v *View) selectBottomTab(title string) bool {
	if v.bottomTabs == nil {
		return false
	}
	for _, item := range v.bottomTabs.Items {
		if strings.EqualFold(item.Text, title) {
			v.bottomTabs.Select(item)
			return true
		}
	}
	return false
}

func (v *View) isBottomTabSelected(title string) bool {
	if v.bottomTabs == nil {
		return false
	}
	selected := v.bottomTabs.Selected()
	if selected == nil {
		return false
	}
	return strings.EqualFold(selected.Text, title)
}
