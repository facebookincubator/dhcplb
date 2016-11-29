name 'dhcplb'
run_list(
  'recipe[dhcplb]',
  'recipe[golang]',
  'recipe[golang::packages]',
)
