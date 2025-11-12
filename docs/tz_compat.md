# Chrono Time Zone Compatibility Wrapper

This repository now includes a header-only wrapper that helps migrate code bases
from Howard Hinnant's `date/tz.h` extension to the C++20 standard
`<chrono>` time zone facilities with minimal source changes.

## Location

- Header: `cpp/include/date/tz_compat.hpp`

Add your compiler's include path so that existing source files that used to rely
on `date::` time zone utilities can include the new wrapper instead.

```cpp
#include "date/tz_compat.hpp"
```

## Provided Compatibility Surface

The wrapper forwards the most widely used time zone APIs to their `<chrono>`
equivalents, including:

- Aliases for key types (`date::zoned_time`, `date::time_zone`, `date::sys_time`, …)
- Accessors (`date::current_zone`, `date::locate_zone`, `date::get_tzdb`)
- Conversion helpers (`date::to_sys`, `date::to_local`) for both zone objects and
  zone names
- Optional helpers such as `date::current_zone_ref()` to obtain a safe reference

These aliases allow the vast majority of existing `date::` time zone usage to
compile unchanged while taking advantage of the standard library implementation.

## Usage Example

```cpp
#include "date/tz_compat.hpp"

int main() {
    using namespace std::chrono;

    auto ny = date::locate_zone("America/New_York");
    auto now = floor<seconds>(system_clock::now());
    date::zoned_time zt{ny, now};

    auto utc = date::to_sys(zt.get_local_time(), "UTC");
    (void)utc;
}
```

## Requirements

- A C++20 (or later) compiler and standard library implementation that ships the
  time zone extensions (`__cpp_lib_chrono >= 201907L`).
- An available time zone database for the standard library on your target system.

## Notes

- The wrapper is intentionally lightweight; if your project relies on rarely used
  `tz.h` helpers that are not yet covered, you can extend
  `cpp/include/date/tz_compat.hpp` with additional aliases or forwarding
  functions that follow the same pattern.
- To remove the wrapper at a later time, update your includes to the standard
  headers and replace `date::` qualifiers with `std::chrono::`.

