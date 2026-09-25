# Apache-2.0 adoption

The user accepted the recommendation to license Teleskope under Apache-2.0 and
explicitly requested applying it to the repository. Add the unmodified official
license text, a README license declaration, and explicit release archive inclusion.
Require the license in archive verification so future releases cannot silently
omit it. Update PH readiness to distinguish this local adoption from older
published binaries and the previous remote audit. Third-party components retain
their original terms; this change does not claim a full dependency license audit.

Validate the official text, archive verification success/failure for ZIP and
TAR.GZ packages, release configuration, normal repository checks and build.
Do not publish a release or push as part of this request.
