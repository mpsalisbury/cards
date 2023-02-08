load("@bazel_gazelle//:deps.bzl", "go_repository")

def go_dependencies():
    go_repository(
        name = "com_github_google_go_cmp",
        importpath = "github.com/google/go-cmp",
        sum = "h1:e6P7q2lk1O+qJJb4BtCQXlK8vWEO8V1ZeuEdJNOqZyg=",
        version = "v0.5.8",
    )
    go_repository(
        name = "org_golang_x_exp",
        importpath = "golang.org/x/exp",
        sum = "h1:BEABXpNXLEz0WxtA+6CQIz2xkg80e+1zrhWyMcq8VzE=",
        version = "v0.0.0-20230131160201-f062dba9d201",
    )
    go_repository(
        name = "org_golang_x_mod",
        importpath = "golang.org/x/mod",
        sum = "h1:b9gGHsz9/HhJ3HF5DHQytPpuwocVTChQJK3AvoLRD5I=",
        version = "v0.6.0",
    )
    go_repository(
        name = "org_golang_x_sys",
        importpath = "golang.org/x/sys",
        sum = "h1:kunALQeHf1/185U1i0GOB/fy1IPRDDpuoOOqRReG57U=",
        version = "v0.1.0",
    )
    go_repository(
        name = "org_golang_x_tools",
        importpath = "golang.org/x/tools",
        sum = "h1:G6AHpWxTMGY1KyEYoAQ5WTtIekUUvDNjan3ugu60JvE=",
        version = "v0.2.0",
    )
