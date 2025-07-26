import { useTranslation } from 'react-i18next';

import { Iconify } from 'src/components/iconify';

import type { AccountPopoverProps } from './components/account-popover';

// ----------------------------------------------------------------------

export const useAccountConfig = (): AccountPopoverProps['data'] => {
  const { t } = useTranslation();

  return [
    {
      label: t('navigation.home'),
      href: '/',
      icon: <Iconify width={22} icon="solar:home-angle-bold-duotone" />,
    },
    {
      label: t('navigation.profile'),
      href: '/profile',
      icon: <Iconify width={22} icon="solar:shield-keyhole-bold-duotone" />,
    },
    {
      label: t('navigation.settings'),
      href: '#',
      icon: <Iconify width={22} icon="solar:settings-bold-duotone" />,
    },
  ];
};

// 保持向后兼容性的导出
export const _account: AccountPopoverProps['data'] = [];
