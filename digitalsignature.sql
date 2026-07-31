--
-- PostgreSQL database dump
--

\restrict ENa5tYqMeDm2OmBPZjW5mcssgKtNYtVNaOge2n8stHDNNvjaiN1dgoDFqvhGLk8

-- Dumped from database version 18.4 (Debian 18.4-1.pgdg13+1)
-- Dumped by pg_dump version 18.4

-- Started on 2026-07-16 14:00:08

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- TOC entry 222 (class 1259 OID 16407)
-- Name: dokumen; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.dokumen (
    id integer NOT NULL,
    user_id integer NOT NULL,
    judul character varying(255) NOT NULL,
    deskripsi text,
    pesan text,
    tipe character varying(100),
    jenis character varying(50) NOT NULL,
    file_path character varying(255) NOT NULL,
    final_file_path character varying(255),
    status character varying(50) DEFAULT 'draft'::character varying,
    created_at timestamp without time zone DEFAULT now()
);


ALTER TABLE public.dokumen OWNER TO postgres;

--
-- TOC entry 221 (class 1259 OID 16406)
-- Name: dokumen_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.dokumen_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.dokumen_id_seq OWNER TO postgres;

--
-- TOC entry 3518 (class 0 OID 0)
-- Dependencies: 221
-- Name: dokumen_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.dokumen_id_seq OWNED BY public.dokumen.id;


--
-- TOC entry 232 (class 1259 OID 16527)
-- Name: log_aktivitas; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.log_aktivitas (
    id integer NOT NULL,
    user_id integer NOT NULL,
    dokumen_id integer,
    aksi character varying(100) NOT NULL,
    keterangan text,
    created_at timestamp without time zone DEFAULT now()
);


ALTER TABLE public.log_aktivitas OWNER TO postgres;

--
-- TOC entry 231 (class 1259 OID 16526)
-- Name: log_aktivitas_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.log_aktivitas_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.log_aktivitas_id_seq OWNER TO postgres;

--
-- TOC entry 3519 (class 0 OID 0)
-- Dependencies: 231
-- Name: log_aktivitas_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.log_aktivitas_id_seq OWNED BY public.log_aktivitas.id;


--
-- TOC entry 224 (class 1259 OID 16428)
-- Name: permintaan_ttd; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.permintaan_ttd (
    id integer NOT NULL,
    dokumen_id integer NOT NULL,
    user_id integer NOT NULL,
    urutan integer NOT NULL,
    page_number integer NOT NULL,
    koordinat_x double precision,
    koordinat_y double precision,
    width double precision,
    height double precision,
    status character varying(50) DEFAULT 'menunggu'::character varying,
    alasan_tolak text,
    created_at timestamp without time zone DEFAULT now()
);


ALTER TABLE public.permintaan_ttd OWNER TO postgres;

--
-- TOC entry 223 (class 1259 OID 16427)
-- Name: permintaan_ttd_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.permintaan_ttd_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.permintaan_ttd_id_seq OWNER TO postgres;

--
-- TOC entry 3520 (class 0 OID 0)
-- Dependencies: 223
-- Name: permintaan_ttd_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.permintaan_ttd_id_seq OWNED BY public.permintaan_ttd.id;


--
-- TOC entry 228 (class 1259 OID 16471)
-- Name: sertifikat; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.sertifikat (
    id integer NOT NULL,
    user_id integer NOT NULL,
    serial_number character varying(255) NOT NULL,
    public_key text NOT NULL,
    status character varying(50) DEFAULT 'active'::character varying,
    valid_from timestamp without time zone,
    valid_until timestamp without time zone,
    created_at timestamp without time zone DEFAULT now()
);


ALTER TABLE public.sertifikat OWNER TO postgres;

--
-- TOC entry 227 (class 1259 OID 16470)
-- Name: sertifikat_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.sertifikat_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.sertifikat_id_seq OWNER TO postgres;

--
-- TOC entry 3521 (class 0 OID 0)
-- Dependencies: 227
-- Name: sertifikat_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.sertifikat_id_seq OWNED BY public.sertifikat.id;


--
-- TOC entry 226 (class 1259 OID 16454)
-- Name: tanda_tangan; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.tanda_tangan (
    id integer NOT NULL,
    user_id integer NOT NULL,
    tipe character varying(50) NOT NULL,
    file_path character varying(255) NOT NULL,
    created_at timestamp without time zone DEFAULT now()
);


ALTER TABLE public.tanda_tangan OWNER TO postgres;

--
-- TOC entry 225 (class 1259 OID 16453)
-- Name: tanda_tangan_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.tanda_tangan_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.tanda_tangan_id_seq OWNER TO postgres;

--
-- TOC entry 3522 (class 0 OID 0)
-- Dependencies: 225
-- Name: tanda_tangan_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.tanda_tangan_id_seq OWNED BY public.tanda_tangan.id;


--
-- TOC entry 230 (class 1259 OID 16493)
-- Name: transaksi_sertifikat; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.transaksi_sertifikat (
    id integer NOT NULL,
    permintaan_ttd_id integer,
    dokumen_id integer NOT NULL,
    user_id integer NOT NULL,
    sertifikat_id integer,
    aksi character varying(50) NOT NULL,
    alasan_tolak text,
    file_result_path character varying(255),
    created_at timestamp without time zone DEFAULT now()
);


ALTER TABLE public.transaksi_sertifikat OWNER TO postgres;

--
-- TOC entry 229 (class 1259 OID 16492)
-- Name: transaksi_sertifikat_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.transaksi_sertifikat_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.transaksi_sertifikat_id_seq OWNER TO postgres;

--
-- TOC entry 3523 (class 0 OID 0)
-- Dependencies: 229
-- Name: transaksi_sertifikat_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.transaksi_sertifikat_id_seq OWNED BY public.transaksi_sertifikat.id;


--
-- TOC entry 220 (class 1259 OID 16390)
-- Name: users; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.users (
    id integer NOT NULL,
    name character varying(255) NOT NULL,
    email character varying(255) NOT NULL,
    password character varying(255) NOT NULL,
    role character varying(50) DEFAULT 'user'::character varying,
    created_at timestamp without time zone DEFAULT now()
);


ALTER TABLE public.users OWNER TO postgres;

--
-- TOC entry 219 (class 1259 OID 16389)
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.users_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.users_id_seq OWNER TO postgres;

--
-- TOC entry 3524 (class 0 OID 0)
-- Dependencies: 219
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;


--
-- TOC entry 3322 (class 2604 OID 16410)
-- Name: dokumen id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.dokumen ALTER COLUMN id SET DEFAULT nextval('public.dokumen_id_seq'::regclass);


--
-- TOC entry 3335 (class 2604 OID 16530)
-- Name: log_aktivitas id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.log_aktivitas ALTER COLUMN id SET DEFAULT nextval('public.log_aktivitas_id_seq'::regclass);


--
-- TOC entry 3325 (class 2604 OID 16431)
-- Name: permintaan_ttd id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.permintaan_ttd ALTER COLUMN id SET DEFAULT nextval('public.permintaan_ttd_id_seq'::regclass);


--
-- TOC entry 3330 (class 2604 OID 16474)
-- Name: sertifikat id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.sertifikat ALTER COLUMN id SET DEFAULT nextval('public.sertifikat_id_seq'::regclass);


--
-- TOC entry 3328 (class 2604 OID 16457)
-- Name: tanda_tangan id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.tanda_tangan ALTER COLUMN id SET DEFAULT nextval('public.tanda_tangan_id_seq'::regclass);


--
-- TOC entry 3333 (class 2604 OID 16496)
-- Name: transaksi_sertifikat id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.transaksi_sertifikat ALTER COLUMN id SET DEFAULT nextval('public.transaksi_sertifikat_id_seq'::regclass);


--
-- TOC entry 3319 (class 2604 OID 16393)
-- Name: users id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);


--
-- TOC entry 3342 (class 2606 OID 16421)
-- Name: dokumen dokumen_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.dokumen
    ADD CONSTRAINT dokumen_pkey PRIMARY KEY (id);


--
-- TOC entry 3354 (class 2606 OID 16538)
-- Name: log_aktivitas log_aktivitas_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.log_aktivitas
    ADD CONSTRAINT log_aktivitas_pkey PRIMARY KEY (id);


--
-- TOC entry 3344 (class 2606 OID 16442)
-- Name: permintaan_ttd permintaan_ttd_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.permintaan_ttd
    ADD CONSTRAINT permintaan_ttd_pkey PRIMARY KEY (id);


--
-- TOC entry 3348 (class 2606 OID 16484)
-- Name: sertifikat sertifikat_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.sertifikat
    ADD CONSTRAINT sertifikat_pkey PRIMARY KEY (id);


--
-- TOC entry 3350 (class 2606 OID 16486)
-- Name: sertifikat sertifikat_serial_number_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.sertifikat
    ADD CONSTRAINT sertifikat_serial_number_key UNIQUE (serial_number);


--
-- TOC entry 3346 (class 2606 OID 16464)
-- Name: tanda_tangan tanda_tangan_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.tanda_tangan
    ADD CONSTRAINT tanda_tangan_pkey PRIMARY KEY (id);


--
-- TOC entry 3352 (class 2606 OID 16505)
-- Name: transaksi_sertifikat transaksi_sertifikat_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.transaksi_sertifikat
    ADD CONSTRAINT transaksi_sertifikat_pkey PRIMARY KEY (id);


--
-- TOC entry 3338 (class 2606 OID 16405)
-- Name: users users_email_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_email_key UNIQUE (email);


--
-- TOC entry 3340 (class 2606 OID 16403)
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- TOC entry 3355 (class 2606 OID 16422)
-- Name: dokumen dokumen_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.dokumen
    ADD CONSTRAINT dokumen_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- TOC entry 3364 (class 2606 OID 16544)
-- Name: log_aktivitas log_aktivitas_dokumen_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.log_aktivitas
    ADD CONSTRAINT log_aktivitas_dokumen_id_fkey FOREIGN KEY (dokumen_id) REFERENCES public.dokumen(id);


--
-- TOC entry 3365 (class 2606 OID 16539)
-- Name: log_aktivitas log_aktivitas_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.log_aktivitas
    ADD CONSTRAINT log_aktivitas_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- TOC entry 3356 (class 2606 OID 16443)
-- Name: permintaan_ttd permintaan_ttd_dokumen_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.permintaan_ttd
    ADD CONSTRAINT permintaan_ttd_dokumen_id_fkey FOREIGN KEY (dokumen_id) REFERENCES public.dokumen(id);


--
-- TOC entry 3357 (class 2606 OID 16448)
-- Name: permintaan_ttd permintaan_ttd_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.permintaan_ttd
    ADD CONSTRAINT permintaan_ttd_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- TOC entry 3359 (class 2606 OID 16487)
-- Name: sertifikat sertifikat_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.sertifikat
    ADD CONSTRAINT sertifikat_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- TOC entry 3358 (class 2606 OID 16465)
-- Name: tanda_tangan tanda_tangan_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.tanda_tangan
    ADD CONSTRAINT tanda_tangan_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- TOC entry 3360 (class 2606 OID 16511)
-- Name: transaksi_sertifikat transaksi_sertifikat_dokumen_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.transaksi_sertifikat
    ADD CONSTRAINT transaksi_sertifikat_dokumen_id_fkey FOREIGN KEY (dokumen_id) REFERENCES public.dokumen(id);


--
-- TOC entry 3361 (class 2606 OID 16506)
-- Name: transaksi_sertifikat transaksi_sertifikat_permintaan_ttd_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.transaksi_sertifikat
    ADD CONSTRAINT transaksi_sertifikat_permintaan_ttd_id_fkey FOREIGN KEY (permintaan_ttd_id) REFERENCES public.permintaan_ttd(id);


--
-- TOC entry 3362 (class 2606 OID 16521)
-- Name: transaksi_sertifikat transaksi_sertifikat_sertifikat_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.transaksi_sertifikat
    ADD CONSTRAINT transaksi_sertifikat_sertifikat_id_fkey FOREIGN KEY (sertifikat_id) REFERENCES public.sertifikat(id);


--
-- TOC entry 3363 (class 2606 OID 16516)
-- Name: transaksi_sertifikat transaksi_sertifikat_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.transaksi_sertifikat
    ADD CONSTRAINT transaksi_sertifikat_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


-- Completed on 2026-07-16 14:00:26

--
-- PostgreSQL database dump complete
--

\unrestrict ENa5tYqMeDm2OmBPZjW5mcssgKtNYtVNaOge2n8stHDNNvjaiN1dgoDFqvhGLk8