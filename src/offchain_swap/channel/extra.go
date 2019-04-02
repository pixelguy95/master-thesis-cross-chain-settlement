package channel

func (c *Channel) isLiteCoinChannel() bool {
	if c.Party1.IsLitecoinUser && c.Party2.IsLitecoinUser {
		return true
	}

	return true
}
