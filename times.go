package main

import (
	util "github.com/PeterXu/goutil"
)

type TimeInfo struct {
	utime int64 // update time
	ctime int64 // create time
}

func NewTimeInfo() *TimeInfo {
	now := util.NowMs64()
	return &TimeInfo{
		utime: now,
		ctime: now,
	}
}

func (ti *TimeInfo) updateTime() {
	ti.utime = util.NowMs64()
}

func (ti *TimeInfo) isTimeout(timeout int) bool {
	return util.NowMs64() >= (ti.utime + int64(timeout))
}

func (ti *TimeInfo) sinceLastUpdate() int {
	return int(util.NowMs64() - ti.utime)
}
