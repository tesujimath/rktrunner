Name:           rktrunner
Version:        %(echo $VERSION)
Release:        1%{?dist}
Summary:        Enable unprivileged users to run containers using rkt

Group:          utils
License:        APLv2
URL:            https://github.com/tesujimath/rktrunner
# pesky github, download URL does not end in the filename they give you
#Source0:        https://github.com/tesujimath/rktrunner/archive/v%{version}.tar.gz
Source0:        rktrunner-%{version}.tar.gz
Source1:        golang.org.x.sys.unix.tgz
BuildRoot:      %{_tmppath}/%{name}-%{version}-%{release}-root-%(%{__id_u} -n)

%description
This package provides the rkt-run command, which is intended to be
installed setuid root, to enabled unprivileged users to run containers
using rkt, in a controlled fashion.

All rkt run options are controlled by the config file,
/etc/rktrunner.toml, which should be carefully setup by the local
sysadmin, perhaps based on the examples which may be found in
%{_datadir}/doc/%{name}-%{version}.

%prep
%setup -q -c
%global gopath %{_builddir}/%{name}-%{version}/go
%global packagehome %{gopath}/src/github.com/tesujimath/%{name}
mkdir -p %{packagehome}
mv %{name}-%{version}/* %{packagehome}
mv %{packagehome}/{LICENSE,README.md,examples} .
tar xaf %SOURCE1 -C %{gopath}/src

%define debug_package %{nil}

%build
cd %{packagehome}
GOPATH=%{gopath} dep ensure
GOPATH=%{gopath} make %{?_smp_mflags}


%install
rm -rf %{buildroot}
mkdir -p %{buildroot}%{_bindir}
mkdir -p %{buildroot}%{_sbindir}
mkdir -p %{buildroot}%{_libexecdir}/%{name}
mkdir -p %{buildroot}%{_mandir}/man1 %{buildroot}%{_mandir}/man5

install -m 0755 %{gopath}/bin/rkt-run %{buildroot}%{_bindir}
install -m 0755 %{gopath}/bin/rktrunner-gc %{buildroot}%{_sbindir}
install -m 0755 %{gopath}/bin/rkt-run-helper %{buildroot}%{_libexecdir}/%{name}
install -m 0755 %{gopath}/bin/rkt-run-slave %{buildroot}%{_libexecdir}/%{name}
install -m 0644 %{packagehome}/doc/rkt-run.1.gz %{buildroot}%{_mandir}/man1
install -m 0644 %{packagehome}/doc/rktrunner.toml.5.gz %{buildroot}%{_mandir}/man5

%clean
rm -rf %{buildroot}


%files
%defattr(-,root,root,-)
%doc LICENSE README.md examples/*
%{_mandir}/man1/*
%{_mandir}/man5/*
%attr(04755,root,root) %{_bindir}/rkt-run
%{_sbindir}/*
%{_libexecdir}/%{name}

%changelog
* Tue Apr 18 2017 Simon Guest <simon.guest@tesujimath.org>
- revised packaging with version as an rpmbuild parameter
- manpages built by Makefile
