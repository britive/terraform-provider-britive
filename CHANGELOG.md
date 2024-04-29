## 2.0.9

ENHANCEMENTS:

* resource/britive_profile: Include AccountID as an association value for AWS Standalone applications (PAB-15749)

BUG FIXES:

* resource/britive_profile_permission: The terraform plan fails after the creation of a profile permission with a "/" in its name (PAB-15794)