package go_time_macro

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// ExpandTimeMacro 将 ${DATE} 等时间相关等宏展开
// 变量基本运算法则：往前或者往后n个单位时间，如：
// ${DATE+n} or ${DATE-n} ：以DATE为锚点往后(+)或者向前(-)n天
// ${hour+n} or ${hour-n}：以hour为锚点往后(+)或者向前(-)n小时
//
//	变量高级运算法则：${var+x+ym-zd+ph-qs}表示业务时间加x单位时间（如果var=date，加x天），
//
// 加y个月，减z天，加p小时，减q秒，最后按照var的输出格式输出，如：
// 业务时间=2018-01-01，${date+1+2m}=20180302，${date-1m}=20171201
// 月级别任务常量配置推荐使用${var-1m}，可支持跨年场景；
// 获取任意月份最后一天：利用${last_date+N}、${last_DATE+N}和${last_day+N} 可获取任意月份的最后一天，以2019-02-21执行结果为例：
// ${last_DATE} = 2019-02-28
// ${last_DATE-1} = 2019-01-31
// ${last_date+1} = 20190331
// ${last_day} = 28
func ExpandTimeMacro(rawSQL string, t time.Time) string {
	result := macroRegex.ReplaceAllStringFunc(rawSQL, func(match string) string {
		h, err := parseMacro(match)
		if err != nil {
			return match
		}

		offset := h.offsetTime(t)
		switch h.name {
		case dateHyperMacro:
			return offset.Format("2006-01-02")
		case dateMacro:
			return offset.Format("20060102")
		case hourXMacro:
			return strconv.Itoa(offset.Hour())
		case hourHHMacro:
			return fmt.Sprintf("%02d", offset.Hour())
		case dayDDMacro:
			return strconv.Itoa(offset.Day())
		case monthMacro:
			return fmt.Sprintf("%02d", int(offset.Month()))
		case timestampMacro:
			return strconv.FormatInt(offset.Unix(), 10)
		case weekOfYearMacro:
			return fmt.Sprintf("%02d", offset.Weekday())
		}
		return match
	})

	return result
}

const (
	dateHyperMacro  = "DATE"         // ${DATE} 业务时间日期，格式为:yyyy-mm-dd，如:2015-05-17
	dateMacro       = "date"         // ${date} 业务时间日期，格式为:yyyymmdd，如:20150526
	hourXMacro      = "HOUR"         // ${HOUR} 业务时间整点，用于小时级别任务，格式为:x（整数），如：2
	hourHHMacro     = "hour"         // ${hour} 业务时间整点，用于小时级别任务，格式为:hh，如：02
	dayDDMacro      = "day"          // ${day} 业务时间日期，用于天级别任务，格式为:dd，如：15
	monthMacro      = "month"        // ${month} 业务时间月份，用于月级别任务，格式为:mm，如：03
	timestampMacro  = "timestamp"    // ${timestamp} 业务时间时间戳， 格式为:x（整数），使用前请核对是否符合预期
	weekOfYearMacro = "week_of_year" // ${week_of_year} 当前时间是本年的第几周， 格式为:%02d（01~52）
)

const (
	groupNameVar          = "var"
	groupNameOffset       = "offset"
	groupNameOffsetMonth  = "offsetMonth"
	groupNameOffsetDay    = "offsetDay"
	groupNameOffsetHour   = "offsetHour"
	groupNameOffsetSecond = "offsetSecond"
)

var macroRegex = regexp.MustCompile(`\${(?P<var>DATE|date|hour|day|month|timestamp|week_of_year)(?P<offset>[+\-]\d+)?((?P<offsetMonth>[+\-]\d+)m)?((?P<offsetDay>[+\-]\d+)d)?((?P<offsetHour>[+\-]\d+)h)?((?P<offsetSecond>[+\-]\d+)s)?}`)

type macroHandler struct {
	name         string
	offset       *int
	offsetMonth  *int
	offsetDate   *int
	offsetHour   *int
	offsetSecond *int

	err error
}

func parseMacro(match string) (*macroHandler, error) {
	matches := macroRegex.FindStringSubmatch(match)

	paramsMap := make(map[string]string)
	for i, name := range macroRegex.SubexpNames() {
		if i > 0 && i <= len(match) {
			paramsMap[name] = matches[i]
		}
	}

	macroName, ok := paramsMap[groupNameVar]
	if !ok {
		return nil, errors.New("cannot find var")
	}

	h := macroHandler{name: macroName}
	h.offset = h.parseGroup(paramsMap, groupNameOffset)
	h.offsetMonth = h.parseGroup(paramsMap, groupNameOffsetMonth)
	h.offsetDate = h.parseGroup(paramsMap, groupNameOffsetDay)
	h.offsetHour = h.parseGroup(paramsMap, groupNameOffsetHour)
	h.offsetSecond = h.parseGroup(paramsMap, groupNameOffsetSecond)

	if h.err != nil {
		return nil, h.err
	}

	return &h, nil
}

func (m *macroHandler) parseGroup(paramsMap map[string]string, groupName string) *int {
	if m.err != nil {
		return nil
	}

	offsetString, ok := paramsMap[groupName]
	if !ok {
		return nil
	}
	if offsetString == "" {
		return nil
	}
	offsetInt64, err := strconv.ParseInt(offsetString, 10, 64)
	if err != nil {
		m.err = fmt.Errorf("parse %s error: %w", groupName, err)
		return nil
	}
	offset := int(offsetInt64)
	return &offset
}

func (m *macroHandler) offsetTime(t time.Time) time.Time {
	if m.offset != nil {
		switch m.name {
		case dateMacro, dateHyperMacro, dayDDMacro:
			t = t.AddDate(0, 0, *m.offset)
		case hourXMacro, hourHHMacro:
			t = t.Add(time.Duration(*m.offset) * time.Hour)
		case monthMacro:
			t = t.AddDate(0, *m.offset, 0)
		case timestampMacro:
			t = t.Add(time.Duration(*m.offset) * time.Second)
		}
	}

	if m.offsetMonth != nil {
		t = t.AddDate(0, *m.offsetMonth, 0)
	}
	if m.offsetDate != nil {
		t = t.AddDate(0, 0, *m.offsetDate)
	}
	if m.offsetHour != nil {
		t = t.Add(time.Duration(*m.offsetHour) * time.Hour)
	}
	if m.offsetSecond != nil {
		t = t.Add(time.Duration(*m.offsetSecond) * time.Second)
	}
	return t
}
