export default () => {
  return {
    computed: {
      rules() {
        const minChars = 8;
        const requiredMessage = this.$vuetify.locale.t('$vuetify.required');
        const minCharsMessage = this.$vuetify.locale.t('$vuetify.min_characters', minChars);
        const invalidEmailMessage = this.$vuetify.locale.t('$vuetify.invalid_email');
        return {
          required: value => !!value || requiredMessage,
          min: v => v.length >= minChars || minCharsMessage,
          email: value => {
            const pattern = /^(([^<>()[\]\\.,;:\s@"]+(\.[^<>()[\]\\.,;:\s@"]+)*)|(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$/
            return pattern.test(value) || invalidEmailMessage
          },
        }
      },
    }
  }
}
