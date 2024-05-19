package di

type Parameter struct {
	qualifier bool
}

func (p *Parameter) IsQualifier() bool {
	return p.qualifier
}
