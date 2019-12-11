create index if not exists idx_deposits_per_member
    on deposits (member_ref);

create index if not exists idx_deposits_per_state
    on deposits (deposit_state);

alter table deposits alter column member_ref drop not null;
