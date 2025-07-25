import { useCallback } from 'react';
import { useTranslation } from 'react-i18next';

import { Box, Button, Stack, Typography } from '@mui/material';

import { useAuth } from 'src/contexts/auth-context';
import { Iconify } from 'src/components/iconify';

// ----------------------------------------------------------------------

interface OAuthButtonsProps {
  disabled?: boolean;
}

export function OAuthButtons({ disabled = false }: OAuthButtonsProps) {
  const { t } = useTranslation();
  const { oauthLogin, state } = useAuth();

  const handleGoogleLogin = useCallback(async () => {
    try {
      await oauthLogin('google');
    } catch (error) {
      console.error('Google login failed:', error);
    }
  }, [oauthLogin]);

  const handleGitHubLogin = useCallback(async () => {
    try {
      await oauthLogin('github');
    } catch (error) {
      console.error('GitHub login failed:', error);
    }
  }, [oauthLogin]);

  return (
    <Stack
      direction={{ xs: 'column', sm: 'row' }}
      spacing={2}
      sx={{ width: '100%' }}
    >
      <Button
        size="large"
        color="inherit"
        variant="outlined"
        disabled={disabled || state.isLoading}
        startIcon={<Iconify icon="socials:google" />}
        onClick={handleGoogleLogin}
        sx={{
          flex: 1,
          borderColor: 'divider',
          '&:hover': {
            borderColor: 'text.primary',
            backgroundColor: 'action.hover',
          },
        }}
      >
        <Typography variant="body2" sx={{ color: 'text.primary' }}>
          {t('auth.continue_with_google')}
        </Typography>
      </Button>

      <Button
        size="large"
        color="inherit"
        variant="outlined"
        disabled={disabled || state.isLoading}
        startIcon={<Iconify icon="socials:github" />}
        onClick={handleGitHubLogin}
        sx={{
          flex: 1,
          borderColor: 'divider',
          '&:hover': {
            borderColor: 'text.primary',
            backgroundColor: 'action.hover',
          },
        }}
      >
        <Typography variant="body2" sx={{ color: 'text.primary' }}>
          {t('auth.continue_with_github')}
        </Typography>
      </Button>
    </Stack>
  );
}
