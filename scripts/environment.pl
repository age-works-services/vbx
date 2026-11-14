#!/usr/bin/env perl
# Parse dotenv-style files as data. Never evaluate shell expressions.
use strict;
use warnings;
my %values;
for my $file (@ARGV) {
    open my $fh, '<', $file or die "$file: cannot read environment file\n";
    my $line_number = 0;
    while (my $line = <$fh>) {
        ++$line_number;
        chomp $line; $line =~ s/\r$//;
        next if $line =~ /^\s*(?:#.*)?$/;
        my $where = "$file:$line_number";
        $line =~ /^\s*(?:export\s+)?([A-Za-z_][A-Za-z0-9_]*)=(.*)$/
            or die "$where: expected NAME=VALUE\n";
        my ($key, $value) = ($1, $2);
        $key !~ /^(?:HOME|USER|LOGNAME|SHELL|PATH|BASH_ENV|ENV|VBX_.*|XDG_.*)$/
            or die "$where: reserved environment variable $key\n";
        $value =~ s/^\s+|\s+$//g;
        if ($value =~ /^'([^']*)'\s*(?:#.*)?$/ || $value =~ /^"([^"\\]*)"\s*(?:#.*)?$/) {
            $value = $1;
        } elsif ($value =~ /^['"]/) {
            die "$where: use a single-line literal value without escaped quotes\n";
        } else { $value =~ s/\s+#.*$//; }
        $value !~ /[\x00-\x1f\x7f]/ or die "$where: control characters are unsupported\n";
        $values{$key} = $value;
    }
    close $fh;
}
for my $key (sort keys %values) {
    my $value = $values{$key};
    $value =~ s/'/'"'"'/g;
    print "export $key='$value'\n";
}
