#!/usr/bin/env perl
# Validate a deliberately small subset of Debian dependency expressions.
use strict;
use warnings;
sub normalize_apt {
    my ($value, $where) = @_;
    $value =~ s/^\s+|\s+$//g;
    $value =~ /^([a-z0-9][a-z0-9+.-]*(?::[a-z0-9][a-z0-9-]*)?)(?:\s+\((>=|<=|=)\s+([0-9][A-Za-z0-9.+:~\-]*)\))?$/
        or die "$where: expected a package name or package (>=|<=|= VERSION)\n";
    return defined $2 ? "$1 ($2 $3)" : $1;
}
unless (caller) {
    my %seen;
    for my $file (@ARGV) {
        open my $fh, '<', $file or die "$file: $!\n";
        my $line_no = 0;
        while (my $line = <$fh>) {
            ++$line_no;
            chomp $line; $line =~ s/\r$//;
            next if $line =~ /^\s*(?:#.*)?$/;
            my $value = normalize_apt($line, "$file:$line_no");
            print "$value\n" unless $seen{$value}++;
        }
    }
}
1;
