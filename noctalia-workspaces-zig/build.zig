const std = @import("std");

pub fn build(b: *std.Build) void {
    const exe = b.addExecutable(.{
        .name = "noctalia-workspaces",
        .root_module = b.createModule(.{
            .root_source_file = b.path("src/main.zig"),
            .target = b.standardTargetOptions(.{}),
            .optimize = b.standardOptimizeOption(.{}),
            .link_libc = true,
        }),
    });
    exe.root_module.linkSystemLibrary("pcre2-8", .{});
    b.installArtifact(exe);
}
