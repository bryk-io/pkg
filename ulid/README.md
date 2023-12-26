# Package `ulid`

Universally Unique Lexicographically Sortable Identifier implementation.

A GUID/UUID can be suboptimal for many use-cases because:

- It isn't the most character efficient way of encoding 128 bits
- UUID v1/v2 is impractical in many environments, as it requires access
  to a unique, stable MAC address
- UUID v3/v5 requires a unique seed and produces randomly distributed IDs,
  which can cause fragmentation in many data structures
- UUID v4 provides no other information than randomness which can cause
  fragmentation in many data structures

A ULID however:

- Case insensitive
- No special characters (URL safe)
- Monotonic sort order (correctly detects and handles the same millisecond)
- Is compatible with UUID/GUID's
- Lexicographically sortable
- Canonically encoded as a 26 character string, as opposed to the 36 character UUID
- 1.21e+24 unique ULIDs per millisecond (1,208,925,819,614,629,174,706,176
    to be exact)
- Uses Crockford's base32 for better efficiency and readability (5 bits per
    character)

## Disclaimer

This package is a simplified version of the original implementation
at: <https://github.com/oklog/ulid>

 Copyright 2016 The Oklog Authors
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

 <http://www.apache.org/licenses/LICENSE-2.0>

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
