# Upstream supervisor integration checks

Run the verified supervisor acquisition workflow manually on the selected diagnostic candidate with `confirm_matrix` enabled. It runs three native jobs (Linux, macOS, and Windows), each limited to 20 minutes. Do not retry automatically. Review the logs, including the Linux release-security and Windows installer boundary checks, before drawing conclusions.
