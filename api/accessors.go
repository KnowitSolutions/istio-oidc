package api

func (aps *AccessPolicyStatus) GetIngress() *AccessPolicyStatusIngress {
	if aps == nil {
		return nil
	} else {
		return &aps.Ingress
	}
}

func (apsi *AccessPolicyStatusIngress) GetSelector() map[string]string {
	if apsi == nil {
		return nil
	} else {
		return apsi.Selector
	}
}
