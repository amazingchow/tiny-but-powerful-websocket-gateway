package extmongo

type FilterOption struct {
	Uid        *string
	StartTime  *int64
	EndTime    *int64
	PageOffset *int32
	PageLimit  *int32
}

func (opt *FilterOption) SetUid(uid string) *FilterOption {
	opt.Uid = &uid
	return opt
}

func (opt *FilterOption) SetStartTime(st int64) *FilterOption {
	opt.StartTime = &st
	return opt
}

func (opt *FilterOption) SetEndTime(ed int64) *FilterOption {
	opt.EndTime = &ed
	return opt
}

func (opt *FilterOption) SetPageOffset(offset int32) *FilterOption {
	opt.PageOffset = &offset
	return opt
}

func (opt *FilterOption) SetPageLimit(limit int32) *FilterOption {
	opt.PageLimit = &limit
	return opt
}
