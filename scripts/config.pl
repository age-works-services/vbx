#!/usr/bin/env perl
# Parse VBX's deliberately restricted YAML schema. Core Perl modules only.
use strict;
use warnings;
use Cwd qw(abs_path);
use File::Basename qw(dirname);
use JSON::PP qw(decode_json);
use FindBin qw($Bin);
require "$Bin/apt-deps.pl";

my ($project, $primary, $local_file, $project_file) = @ARGV;
defined $project_file or die "Usage: config.pl PROJECT PRIMARY LOCAL_CONFIG PROJECT_CONFIG\n";
my %config = (version => '1', cpus => '2', memory => '4GiB', disk => '30GiB', default => 'vscode');
my (@mounts, @modules, @env_files, @apt);

sub error { die "$_[0]: $_[1]\n" }
sub scalar_value {
    my ($text, $where) = @_;
    $text =~ s/^\s+|\s+$//g;
    error($where, 'Control characters are unsupported') if $text =~ /[\x00-\x1f\x7f]/;
    if ($text =~ /^'((?:[^']|'')*)'\s*(?:#.*)?$/) {
        my $value = $1; $value =~ s/''/'/g; return $value;
    }
    if ($text =~ /^("(?:[^"\\]|\\.)*")\s*(?:#.*)?$/) {
        my $value = eval { decode_json($1) };
        error($where, 'Use JSON-compatible double-quoted strings') if $@;
        error($where, 'Control characters are unsupported') if $value =~ /[\x00-\x1f\x7f]/;
        return $value;
    }
    $text =~ s/\s+#.*$//;
    error($where, 'Expected a plain or quoted scalar')
        if $text eq '' || $text =~ /[\x00-\x1f\x7f'"\[\]{}]|:\s|^[!&*|>@`]/;
    return $text;
}

sub read_config {
    my ($file, $base) = @_;
    return unless -e $file;
    -f $file or error($file, 'Expected a regular file');
    open my $fh, '<', $file or die "$file: $!\n";
    my ($section, $item, $line_no) = ('', undef, 0);
    my %seen;
    while (my $line = <$fh>) {
        ++$line_no;
        chomp $line; $line =~ s/\r$//;
        my $where = "$file:$line_no";
        error($where, 'Tabs are unsupported; use two-space indentation') if $line =~ /\t/;
        next if $line =~ /^\s*(?:#.*)?$/;
        if ($line =~ /^([a-zA-Z][a-zA-Z0-9]*):\s*(.*?)\s*$/) {
            my ($key, $value) = ($1, $2);
            error($where, "Duplicate key $key") if $seen{$key}++;
            $section = $key; $item = undef;
            if ($key eq 'version') {
                $config{version} = scalar_value($value, $where);
                error($where, 'Only version 1 is supported') unless $config{version} eq '1';
            } elsif ($key =~ /^(resources|launch|mounts|modules|envFiles|apt)$/) {
                $value =~ s/\s*#.*$//;
                error($where, 'Expected a block mapping/list') unless $value eq '' || ($key =~ /^(mounts|modules|envFiles)$/ && $value eq '[]');
                $section = '' if $value eq '[]';
            } else { error($where, "Unsupported key $key"); }
            next;
        }
        if ($section eq 'apt') {
            if ($line =~ /^  packages:\s*(.*?)\s*$/) {
                my $value = $1; $value =~ s/\s*#.*$//;
                error($where, 'Duplicate key apt.packages') if $seen{'apt.packages'}++;
                error($where, 'Expected a package list') unless $value eq '' || $value eq '[]';
                $item = $value eq '[]' ? undef : 'packages';
            } elsif (defined $item && $line =~ /^    - (.+)$/) {
                push @apt, normalize_apt(scalar_value($1, $where), $where);
            } else { error($where, 'Use apt.packages with four-space indented list items'); }
        } elsif ($section eq 'modules' || $section eq 'envFiles') {
            $line =~ /^  - (.+)$/ or error($where, 'Expected a two-space indented list item');
            my $value = scalar_value($1, $where);
            if ($section eq 'modules') { push @modules, $value; }
            else {
                $value =~ s{^~(?=/|$)}{$ENV{HOME}};
                $value = "$base/$value" unless $value =~ m{^/};
                $value = abs_path($value);
                defined $value && -f $value or error($where, 'Environment file must exist');
                push @env_files, $value;
            }
        } elsif ($section eq 'mounts') {
            if ($line =~ /^  - source:\s*(.+)$/) {
                $item = {source => scalar_value($1, $where), base => $base, where => $where};
                push @mounts, $item;
            } elsif ($line =~ /^    (target|readonly):\s*(.+)$/ && defined $item) {
                my ($key, $value) = ($1, scalar_value($2, $where));
                error($where, "Duplicate mount $key") if exists $item->{$key};
                $item->{$key} = $value;
            } else { error($where, 'Mounts require source, followed by target and optional readonly'); }
        } elsif ($section eq 'resources' || $section eq 'launch') {
            $line =~ /^  ([a-zA-Z]+):\s*(.+)$/ or error($where, 'Expected two-space indentation and a scalar');
            my ($key, $value) = ($1, scalar_value($2, $where));
            my $allowed = $section eq 'resources' ? qr/^(cpus|memory|disk)$/ : qr/^(default|folder)$/;
            error($where, "Unsupported $section key $key") unless $key =~ $allowed;
            error($where, "Duplicate key $section.$key") if $seen{"$section.$key"}++;
            $config{$key} = $value;
        } else { error($where, 'Unsupported YAML syntax'); }
    }
    close $fh;
    error($file, 'version: 1 is required') unless $seen{version};
}

read_config($local_file, dirname($local_file));
read_config($project_file, $project);
$config{cpus} =~ /^[1-9][0-9]*$/ or error('resources.cpus', 'Expected a positive integer');
for my $key ('memory', 'disk') {
    $config{$key} =~ /^[1-9][0-9]*(?:MiB|GiB|TiB)$/ or error("resources.$key", 'Use a positive integer with MiB, GiB, or TiB');
}
$config{default} =~ /^(vscode|terminal|workspace-only)$/ or error('launch.default', 'Use vscode, terminal, or workspace-only');
$config{folder} //= @mounts ? '/workspaces' : $primary;
$config{folder} =~ m{^/} && $config{folder} !~ /[\x00-\x1f\x7f]/
    or error('launch.folder', 'Expected an absolute guest path');
my %targets = ($primary => 1);
for my $mount (@mounts) {
    my $where = $mount->{where};
    defined $mount->{target} or error($where, 'Mount target is required');
    my $target = $mount->{target};
    $target =~ m{^/workspaces/[a-zA-Z0-9_.-]+$} && $target !~ m{/\.\.?$}
        or error($where, 'Mount target must be an immediate child of /workspaces');
    error($where, "Duplicate mount target $target") if $targets{$target}++;
    my $source = $mount->{source};
    $source =~ s{^~(?=/|$)}{$ENV{HOME}};
    $source = "$mount->{base}/$source" unless $source =~ m{^/};
    $source = abs_path($source);
    defined $source && -d $source or error($where, 'Mount source must be an existing directory');
    $source ne '/' && $source !~ /[\x00-\x1f\x7f]/ or error($where, 'Unsupported mount source');
    for my $other ($project, map { $_->{resolved} // () } @mounts) {
        if ($source eq $other || index("$source/", "$other/") == 0 || index("$other/", "$source/") == 0) {
            error($where, 'Host mount sources must not overlap');
        }
    }
    $mount->{resolved} = $source;
    $mount->{readonly} //= 'false';
    $mount->{readonly} =~ /^(true|false)$/ or error($where, 'readonly must be true or false');
}

# Explicit order only. Recipes own their prerequisites; there is no module graph.
my %known = map { $_ => 1 } qw(dev-tools node pi pi-web-fetch);
my (%selected, @ordered);
for my $name (@modules) {
    exists $known{$name} or error('modules', "Unknown bundled module $name");
    push @ordered, $name unless $selected{$name}++;
}

sub emit { print join("\0", @_), "\0"; }
emit($_, $config{$_}) for qw(cpus memory disk default folder);
for my $mount (@mounts) { emit('mount', $mount->{resolved}, $mount->{target}, $mount->{readonly}); }
emit('module', $_) for @ordered;
emit('envFile', $_) for @env_files;
my %apt_seen;
emit('apt', $_) for grep { !$apt_seen{$_}++ } @apt;
