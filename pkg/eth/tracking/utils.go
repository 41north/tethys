package tracking

func blocksByBlockNumber(a, b *Block) bool {
	if a == nil {
		return b == nil
	}
	if b == nil {
		return a == nil
	}
	return a.Number.Cmp(b.Number) < 0
}
