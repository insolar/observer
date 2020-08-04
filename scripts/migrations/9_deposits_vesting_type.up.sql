create type vesting_type as enum ('default', 'linear');

alter table deposits
    add vesting_type vesting_type not null default 'default';

