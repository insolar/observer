update deposits d
set deposit_number = dn.rn
from (select member_ref, deposit_ref, row_number() over (PARTITION BY member_ref order by transfer_date) as rn
      from deposits) as dn
where d.deposit_ref = dn.deposit_ref