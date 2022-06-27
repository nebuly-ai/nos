package controllers

func BoolAddr(b bool) *bool {
	var boolVar bool
	boolVar = b
	return &boolVar
}
