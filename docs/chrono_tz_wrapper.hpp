#pragma once

#if __has_include(<version>)
#  include <version>
#endif

#include <chrono>
#include <format>
#include <string>
#include <string_view>
#include <type_traits>
#include <utility>

#if !defined(__cpp_lib_chrono) || (__cpp_lib_chrono < 201907L)
#  error "chrono_tz_wrapper.hpp requires C++20 chrono time zone support"
#endif

// This header provides a thin compatibility layer over the C++20 <chrono> time-zone
// APIs so that code originally written against Howard Hinnant's date/tz library can
// migrate with minimal source changes.  It intentionally mirrors the commonly used
// names that tz.h exposed in namespace date::, forwarding each of them to the
// equivalent standard library facility.
//
// Usage:
//   #include "chrono_tz_wrapper.hpp"
//
// All existing references to date::... should continue to compile while you stage
// the migration away from tz.h.  Once the migration is complete the wrapper can be
// removed and callers can target std::chrono directly.

namespace date {

using std::chrono::abs;
using std::chrono::ceil;
using std::chrono::choose;
using std::chrono::days;
using std::chrono::floor;
using std::chrono::hh_mm_ss;
using std::chrono::hours;
using std::chrono::is_am;
using std::chrono::is_pm;
using std::chrono::leap_second;
using std::chrono::local_days;
using std::chrono::local_info;
using std::chrono::local_t;
using std::chrono::minutes;
using std::chrono::months;
using std::chrono::round;
using std::chrono::seconds;
using std::chrono::sys_days;
using std::chrono::sys_info;
using std::chrono::sys_seconds;
using std::chrono::time_zone;
using std::chrono::time_zone_link;
using std::chrono::utc_clock;
using std::chrono::year;
using std::chrono::year_month_day;
using std::chrono::year_month_day_last;
using std::chrono::year_month_weekday;
using std::chrono::year_month_weekday_last;

template <class Duration>
using sys_time = std::chrono::sys_time<Duration>;

template <class Duration>
using local_time = std::chrono::local_time<Duration>;

template <class Duration = std::chrono::seconds, class TimeZonePtr = const std::chrono::time_zone*>
using zoned_time = std::chrono::zoned_time<Duration, TimeZonePtr>;

using zoned_seconds = std::chrono::zoned_time<std::chrono::seconds>;

using tzdb = std::chrono::tzdb;
using tzdb_list = std::chrono::tzdb_list;

inline const time_zone* locate_zone(std::string_view name) {
    return std::chrono::get_tzdb().locate_zone(name);
}

inline const time_zone* current_zone() {
    return std::chrono::current_zone();
}

inline const tzdb& get_tzdb() noexcept {
    return std::chrono::get_tzdb();
}

inline const tzdb_list& get_tzdb_list() noexcept {
    return std::chrono::get_tzdb_list();
}

inline const tzdb& reload_tzdb() {
    return std::chrono::reload_tzdb();
}

inline std::string remote_version() {
    return std::chrono::remote_version();
}

template <class... Args>
constexpr auto make_zoned(Args&&... args)
    -> decltype(std::chrono::zoned_time(std::forward<Args>(args)...)) {
    return std::chrono::zoned_time(std::forward<Args>(args)...);
}

template <class... Args>
auto format(Args&&... args)
    -> decltype(std::chrono::format(std::forward<Args>(args)...)) {
    return std::chrono::format(std::forward<Args>(args)...);
}

template <class... Args>
auto parse(Args&&... args)
    -> decltype(std::chrono::parse(std::forward<Args>(args)...)) {
    return std::chrono::parse(std::forward<Args>(args)...);
}

template <class TimeZone, class Duration>
inline auto make_zoned(TimeZone&& tz, local_time<Duration> lt, choose c)
    -> decltype(std::chrono::zoned_time(std::forward<TimeZone>(tz), lt, c)) {
    return std::chrono::zoned_time(std::forward<TimeZone>(tz), lt, c);
}

template <class TimeZone, class TimePoint>
inline auto make_zoned(TimeZone&& tz, TimePoint&& tp)
    -> decltype(std::chrono::zoned_time(std::forward<TimeZone>(tz), std::forward<TimePoint>(tp))) {
    return std::chrono::zoned_time(std::forward<TimeZone>(tz), std::forward<TimePoint>(tp));
}

template <class TimeZone>
inline auto make_zoned(TimeZone&& tz)
    -> decltype(std::chrono::zoned_time(std::forward<TimeZone>(tz))) {
    return std::chrono::zoned_time(std::forward<TimeZone>(tz));
}

}  // namespace date

#endif  // chrono_tz_wrapper_hpp
