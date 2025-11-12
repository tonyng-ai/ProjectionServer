#pragma once

#include <chrono>
#include <stdexcept>
#include <string_view>
#include <type_traits>

#if !defined(__cpp_lib_chrono) || __cpp_lib_chrono < 201907L
#error "tz_compat.hpp requires C++20 <chrono> timezone support (__cpp_lib_chrono >= 201907L)"
#endif

namespace date {

using std::chrono::ambiguous_local_time;
using std::chrono::choose;
using std::chrono::current_zone;
using std::chrono::file_clock;
using std::chrono::gps_clock;
using std::chrono::leap_second;
using std::chrono::locate_zone;
using std::chrono::local_days;
using std::chrono::local_info;
using std::chrono::local_seconds;
using std::chrono::local_time;
using std::chrono::make_zoned;
using std::chrono::nonexistent_local_time;
using std::chrono::sys_days;
using std::chrono::sys_info;
using std::chrono::sys_seconds;
using std::chrono::sys_time;
using std::chrono::tai_clock;
using std::chrono::time_zone;
using std::chrono::time_zone_link;
using std::chrono::tzdb;
using std::chrono::tzdb_list;
using std::chrono::utc_clock;
using std::chrono::zoned_time;

using std::chrono::get_tzdb;
using std::chrono::get_tzdb_list;
#if __cpp_lib_chrono >= 201907L
using std::chrono::reload_tzdb;
#endif

namespace detail {

inline const std::chrono::time_zone& require_zone(const std::chrono::time_zone* tz)
{
    if (tz == nullptr) {
        throw std::runtime_error{"date::tz_compat: null time zone pointer"};
    }
    return *tz;
}

}  // namespace detail

template <class Duration>
[[nodiscard]] inline std::chrono::sys_time<Duration>
to_sys(const std::chrono::local_time<Duration>& tp,
       const std::chrono::time_zone& tz,
       std::chrono::choose c = std::chrono::choose::earliest)
{
    return tz.to_sys(tp, c);
}

template <class Duration>
[[nodiscard]] inline std::chrono::local_time<Duration>
to_local(const std::chrono::sys_time<Duration>& tp,
         const std::chrono::time_zone& tz)
{
    return tz.to_local(tp);
}

template <class Duration>
[[nodiscard]] inline std::chrono::sys_time<Duration>
to_sys(const std::chrono::local_time<Duration>& tp,
       std::string_view tz_name,
       std::chrono::choose c = std::chrono::choose::earliest)
{
    return detail::require_zone(date::locate_zone(tz_name)).to_sys(tp, c);
}

template <class Duration>
[[nodiscard]] inline std::chrono::sys_time<Duration>
to_sys(const std::chrono::local_time<Duration>& tp,
       const char* tz_name,
       std::chrono::choose c = std::chrono::choose::earliest)
{
    return date::to_sys(tp, std::string_view{tz_name}, c);
}

template <class Duration>
[[nodiscard]] inline std::chrono::local_time<Duration>
to_local(const std::chrono::sys_time<Duration>& tp,
         std::string_view tz_name)
{
    return detail::require_zone(date::locate_zone(tz_name)).to_local(tp);
}

template <class Duration>
[[nodiscard]] inline std::chrono::local_time<Duration>
to_local(const std::chrono::sys_time<Duration>& tp,
         const char* tz_name)
{
    return date::to_local(tp, std::string_view{tz_name});
}

[[nodiscard]] inline const std::chrono::time_zone& current_zone_ref()
{
    return detail::require_zone(date::current_zone());
}

}  // namespace date

