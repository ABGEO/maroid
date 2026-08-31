
    export type RemoteKeys = 'REMOTE_ALIAS_IDENTIFIER/entry';
    type PackageType<T> = T extends 'REMOTE_ALIAS_IDENTIFIER/entry' ? typeof import('REMOTE_ALIAS_IDENTIFIER/entry') :any;