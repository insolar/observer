create type vesting_type as enum ('non-linear', 'vesting-fund', 'linear');

alter table deposits
    add vesting_type vesting_type not null default 'non-linear';
update deposits set vesting_type = 'vesting-fund' where eth_hash = 'genesis_deposit';
