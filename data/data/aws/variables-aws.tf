variable "aws_config_version" {
  description = <<EOF
(internal) This declares the version of the AWS configuration variables.
It has no impact on generated assets but declares the version contract of the configuration.
EOF

  default = "1.0"
}

variable "aws_bootstrap_ec2_type" {
  type        = "string"
  description = "Instance size for the bootstrap node(s). Example: `m4.large`."
  default = "m4.large"
}

variable "aws_master_ec2_type" {
  type        = "string"
  description = "Instance size for the master node(s). Example: `m4.large`."

  # FIXME: get this wired up to the machine default
  default = "m4.xlarge"
}

variable "aws_ec2_ami_override" {
  type        = "string"
  description = "(optional) AMI override for all nodes. Example: `ami-foobar123`."
  default     = ""
}

variable "aws_extra_tags" {
  type = "map"

  description = <<EOF
(optional) Extra AWS tags to be applied to created resources.

Example: `{ "key" = "value", "foo" = "bar" }`
EOF

  default = {}
}

variable "aws_master_root_volume_type" {
  type        = "string"
  default     = "gp2"
  description = "The type of volume for the root block device of master nodes."
}

variable "aws_master_root_volume_size" {
  type        = "string"
  default     = "120"
  description = "The size of the volume in gigabytes for the root block device of master nodes."
}

variable "aws_master_root_volume_iops" {
  type    = "string"
  default = "400"

  description = <<EOF
The amount of provisioned IOPS for the root block device of master nodes.
Ignored if the volume type is not io1.
EOF
}

variable "aws_region" {
  type        = "string"
  description = "The target AWS region for the cluster."
}
